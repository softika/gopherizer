# API package documentation

The `api` package implements the HTTP layer: the server lifecycle, the router
and its middleware stack, the registered operations, and the translation of
domain errors into responses.

Two libraries split the work:

- **[chi](https://go-chi.io/#/README)** owns routing and middleware.
- **[huma](https://huma.rocks)** sits on top and owns request decoding,
  validation, response encoding and the generated OpenAPI document.

Because huma derives the schema from the same struct it binds requests with,
the document cannot drift from the code. There is no spec file to maintain.

---

## Components

### Server (`server.go`)

`Server` manages the lifecycle of the `net/http` server: listening, serving and
shutting down gracefully without interrupting active connections.

Timeouts come from `config.HTTPConfig`:

| Setting | Purpose |
| --- | --- |
| `read_timeout` | Whole-request read deadline |
| `read_header_timeout` | Header read deadline. Defaults to 10s when unset, so the server is never left open to slow-header attacks by omission |
| `write_timeout` | Response write deadline |
| `idle_timeout` | Keep-alive deadline |

`MaxHeaderBytes` is fixed at 1 MiB.

### Router (`router.go`)

`NewRouter` takes an already-connected `database.Service`. The pool is injected
rather than created here so its lifetime is owned by the caller — that is what
makes a clean shutdown possible.

The router is a `chi.Mux` with a `huma.API` mounted on it through the
`humachi` adapter.

#### Middleware order

chi panics if `Use` is called after any route is registered, so every middleware
is installed before the first route. `defaultMiddlewares` therefore registers no
routes itself; it returns the metrics collectors and lets the caller mount the
scrape endpoint afterwards.

The order matters:

1. `tracing`, when enabled — **first**, so the span is on the context that every
   later middleware sees, which is what lets the access log carry a trace id
2. `correlation` — so every later log line carries the request and correlation ids
3. `accessLogger` — registered straight after correlation, so the one line
   recording status and latency is joinable to everything else
4. `middleware.CleanPath`
5. `recoverer` — structured, unlike chi's, which writes plain text to stderr
6. `requestTimeout`, when configured — inside the recoverer, so a panic raised
   while unwinding a timed-out request is still caught
7. `middleware.Heartbeat("/")`
8. `middleware.NoCache`
9. `middleware.AllowContentEncoding("deflate", "gzip")`
10. `bodyLimit`, when configured
11. CORS, when origins are configured
12. metrics, when enabled — inside tracing, so a sampled span is available to
    attach as an exemplar
13. client-IP resolver, then the rate limiter — the resolver must run first or
    the bucket key is wrong

#### OpenAPI configuration

- Document served at `/openapi.json` and `/openapi.yaml`, UI at `/docs`.
- The renderer is SwaggerUI rather than huma's default Stoplight Elements,
  which renders a third-party credit.
- huma's default create hook is dropped. It installs a schema-link transformer
  that stamps a `$schema` field into every response body and a matching `Link`
  header, both embedding the server's own host — deployment detail that does not
  belong in payloads. The generated document is unaffected.

### Operations (`operations.go`)

Every endpoint is registered with `huma.Register`. A registration is the single
source of truth for routing, request decoding, validation and the OpenAPI
operation.

Inputs and outputs are thin HTTP-shaped wrappers around the domain types:

```go
type profileOutput struct {
    Body *profile.Response
}

type createProfileInput struct {
    Body profile.CreateRequest
}
```

Path parameters are constrained with `pattern`, not `format`. JSON Schema
treats `format` as an annotation validators may ignore, while `pattern` is
always enforced — so a malformed id is rejected before it reaches the database:

```go
type profileIdInput struct {
    Id string `path:"id" pattern:"^[0-9a-fA-F]{8}-...-[0-9a-fA-F]{12}$" doc:"Profile identifier"`
}
```

#### Adding an endpoint

1. Add the service method under [`internal/`](../internal/README.md).
2. Declare the request/response types there, with JSON Schema tags for
   validation (`minLength`, `maxLength`, `pattern`, `doc`, `example`).
3. Wrap them in an input/output struct in `operations.go`.
4. Register the operation:

```go
huma.Register(api, huma.Operation{
    OperationID:   "create-profile",
    Method:        http.MethodPost,
    Path:          "/api/v1/profile",
    Summary:       "Create profile",
    Tags:          []string{"Profile"},
    DefaultStatus: http.StatusCreated,
}, func(ctx context.Context, in *createProfileInput) (*profileOutput, error) {
    res, err := svc.Create(ctx, in.Body)
    if err != nil {
        return nil, apiError(ctx, err)
    }
    return &profileOutput{Body: res}, nil
})
```

The route, the validation and the documentation all follow from that one call.

### Error handling (`errors.go`)

Handlers return domain errors; `apiError` logs the full failure server-side —
correlated by the request id the `correlation` middleware put on the context —
and returns a deliberately generic message to the caller. The underlying error
frequently wraps a driver failure carrying table names, hostnames and SQLSTATE
codes, none of which may reach a client.

| `errorx` type | Status | Client message |
| --- | --- | --- |
| `ErrInvalidInput` | 400 | `invalid request` |
| `ErrUnauthorized` | 401 | `unauthorized` |
| `ErrForbidden` | 403 | `forbidden` |
| `ErrNotFound` | 404 | `not found` |
| `ErrUnavailable` | 503 | `service unavailable` |
| `ErrInternal`, or untyped | 500 | `internal server error` |

Two statuses are produced outside this table:

- **422** — huma's own request validation, before a handler is reached. This is
  huma's default for schema violations, not 400.
- **429** — the rate limiter.

### Schema naming (`schema.go`)

huma keeps one global schema registry keyed by type name, and its default namer
ignores the package — so `health.Response` and `profile.Response` collide.
`schemaNamer` qualifies types belonging to this module by their package
(`HealthResponse`, `ProfileResponse`) while leaving foreign types, such as
huma's own error model, under their conventional names.

### Wiring (`bootstrap.go`)

`initRepositories` and `initServices` construct the dependency graph from the
injected `database.Service`. Add new repositories and services to those two
structs.

---

## Middleware

### `correlation.go`

Puts a request id and a correlation id on the request context, where
[slogging](https://github.com/softika/slogging) picks them up and stamps them
onto every record. Both are echoed back so callers can tie their logs to ours.

The two differ in scope: the **correlation id** spans systems and is preserved
when a caller supplies one; the **request id** identifies this hop alone and is
always freshly generated, using `uuid.NewV7` from the standard library so ids
sort chronologically.

An inbound id is accepted only when it is a bounded, printable token. Anything
else is **discarded rather than sanitized** — these values are written to logs,
and a caller must not be able to forge entries by smuggling newlines or quotes
through a header.

### `tracing.go`

Starts a server span per request and joins it to an upstream trace when the
caller sent a `traceparent`.

Written by hand rather than using `otelhttp`, for one reason: `otelhttp` names
the span before chi has routed, which puts the raw path in the span name and
mints one name per path parameter — the same cardinality mistake `routePattern`
exists to prevent for metric labels. Here the span opens under the method alone
and is renamed once the route is known.

Only `5xx` marks a span as an error. A `4xx` is the caller's mistake, and
flagging those would bury real faults.

The tracer provider and propagator are injected rather than read from
OpenTelemetry's globals, so tests can assert on real spans without mutating
process-wide state.

### `accesslog.go`

One structured record per request: method, route pattern, path, status, bytes,
duration, remote address and user agent.

It replaces chi's `middleware.Logger`, which wrote coloured plain text through
the standard library logger. That put two formats on stdout — half of which no
aggregator could parse — and left the one line recording a request's status and
latency with no correlation id, so it could not be joined to the application
logs for the same request.

The level follows the status: `5xx` logs at error, `4xx` at warn, everything
else at info, so a failing endpoint is findable without knowing to filter on a
status field. Requests to the metrics path are skipped; a 15s scrape interval
would otherwise bury the logs this middleware exists to produce.

### `recoverer.go`

Turns a panic into a logged failure and a 500.

It replaces chi's `middleware.Recoverer`, which writes the stack straight to
`os.Stderr` as plain text. That left the most important record in the system in
a format no aggregator could parse and with no request id, so a panic could not
be tied back to the request that caused it — the same defect the access log had,
on the line where it matters most.

Nothing about the panic reaches the caller: a stack names internal paths, types
and sometimes values. The response uses `application/problem+json` with huma's
own error model, so a client parses one error format rather than two.

`http.ErrAbortHandler` is re-panicked rather than reported. It is the standard
library's way of saying "drop this connection deliberately", not a fault.

### `timeout.go`

Gives every request a deadline. Without one a handler is bounded only by the
server's write timeout, which is measured in minutes — long enough for a pool's
worth of slow requests to hold every connection.

It cancels the context rather than writing a response itself, so the handler
sees the cancellation, the query it is waiting on is abandoned, and the error
travels the normal path. `classify` maps a deadline onto `ErrUnavailable`, so a
request that ran out of time answers **503** rather than a misleading 500.

This is the middle of three layers — see
[config/README.md](../config/README.md#layered-timeouts).

### `bodylimit.go`

Caps the request body.

huma already applies a 1 MiB default to operations that read a body, so this is
not the difference between bounded and unbounded for those routes. What it adds
is that the limit is explicit and configurable in one place, and that it covers
routes huma never sees — the heartbeat, the metrics endpoint, anything mounted
on the chi router directly.

The same configured value is passed to the operations that read a body.
Otherwise huma's default would silently overrule a larger configured limit, and
the setting would appear not to work.

### `clientip.go`

Establishes the caller's address for the configured trust model
(`http.client_ip.from`):

| Value | Resolver |
| --- | --- |
| `remote_addr` (default) | Socket peer. Proxy headers ignored |
| `xff` | `X-Forwarded-For`, trusting `trusted_proxies` hops |
| `header` | The proxy-set header named by `trusted_header` |

The default ignores proxy headers on purpose: a caller controls them, so
trusting them without a proxy in front lets anyone forge their identity. Behind
a proxy the opposite failure applies — every client would share the proxy's
address and land in a single rate-limit bucket. The deployment has to say which
case it is; there is no safe guess.

### `ratelimit.go`

Per-client limiting via [httprate](https://github.com/go-chi/httprate), keyed on
the address the resolver established. `CanonicalizeIP` groups IPv6 addresses by
/64 so a client cannot rotate through its own prefix to reset the limit. Setting
`http.rate_limit.requests` or `window` to zero disables it.

### `cors.go`

Built from `http.cors.*`. Credentials are never allowed alongside a wildcard
origin: browsers reject that combination outright, and honouring it would expose
authenticated responses to any site. Narrow `origins` to real hosts before
enabling credentials.

### `metrics.go`

Prometheus collectors on a **private registry**, keeping them out of the global
default so tests can build independent instances without duplicate registration.

Exposes `http_requests_total` and `http_request_duration_seconds`, labelled by
method, route and status, plus the Go and process collectors. The `route` label
is the matched chi route *pattern*, never the raw path — labelling with the raw
path would mint a new time series for every URL a scanner probes, which is a
cheap way to exhaust the metrics backend. Unmatched requests collapse into a
single `unmatched` bucket.

The latency histogram attaches the active `trace_id` as an **exemplar** when the
request was sampled, which is what makes a spike on a dashboard one click from
the trace that caused it. Unsampled requests record latency normally rather than
emitting a link that resolves to nothing. Exemplars require the OpenMetrics
exposition format, so the handler enables it; Prometheus negotiates it via
`Accept` and must be started with `--enable-feature=exemplar-storage` or it will
accept the exemplars and discard them.

### `dbmetrics.go`

Connection pool statistics and build identity.

| Metric | Notes |
| --- | --- |
| `db_pool_connections{state}` | `acquired`, `idle`, `constructing` |
| `db_pool_connections_max` | Pool size |
| `db_pool_acquires_total` | Total acquisitions |
| `db_pool_acquire_waits_total` | Acquisitions that had to wait — the saturation signal |
| `db_pool_acquire_cancels_total` | Acquisitions abandoned before a connection freed |
| `db_pool_acquire_duration_seconds_total` | Total duration of acquires, construction included — not queueing time alone |
| `db_pool_connections_closed_total{reason}` | `idle` or `lifetime` |
| `app_build_info{name,version,environment,go_version}` | Always `1`; the labels are the point |

These numbers were already being computed for the readiness probe and discarded
into a map of strings. Pool saturation is what actually pages someone, and it
was the one signal missing from Prometheus.

The collector reads at scrape time and holds no state, so a scrape reflects the
pool as it is rather than as it was when something last sampled it. A nil pool
yields zeros rather than panicking: collection runs inside a scrape, and a
scrape must never take down the endpoint that explains what is wrong — least of
all while the database is the thing going wrong.

There is deliberately **no** `dependency_up` gauge. It would need a real `Ping`
per scrape, which makes `/metrics` fail when the database does, losing every
metric including the ones diagnosing the outage.
`http_requests_total{route="/health/ready",status="503"}` already carries that
signal.
