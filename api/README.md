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

1. `correlation` — **first**, so every later log line carries the ids
2. `middleware.Logger`
3. `middleware.CleanPath`
4. `middleware.Recoverer`
5. `middleware.Heartbeat("/")`
6. `middleware.NoCache`
7. `middleware.AllowContentEncoding("deflate", "gzip")`
8. CORS, when origins are configured
9. metrics, when enabled
10. client-IP resolver, then the rate limiter — the resolver must run first or
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
