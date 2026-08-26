# Config package documentation

The `config` package manages application configuration. It provides a single
place to load, validate and access settings, sourced from an embedded defaults
file and overridden by environment variables.

---

## Overview

This package uses:

- **[Viper](https://github.com/spf13/viper)** for configuration loading and
  environment binding.
- **[go-playground/validator](https://github.com/go-playground/validator)** to
  ensure required fields are populated.
- An **embedded** [`default.toml`](default.toml) as the baseline, compiled into
  the binary via `//go:embed` — a deployed image needs no config file alongside
  it.

`config.New()` returns `(*Config, error)`. Validation runs at load time, so a
missing or malformed setting fails at startup rather than at first use.

---

## Environment variable overrides

Every key in `default.toml` can be overridden. Uppercase the key path and
replace `.` with `_`:

| TOML key | Environment variable |
| --- | --- |
| `app.name` | `APP_NAME` |
| `app.environment` | `APP_ENVIRONMENT` |
| `app.log_level` | `APP_LOG_LEVEL` |
| `http.port` | `HTTP_PORT` |
| `http.read_header_timeout` | `HTTP_READ_HEADER_TIMEOUT` |
| `http.rate_limit.requests` | `HTTP_RATE_LIMIT_REQUESTS` |
| `http.client_ip.from` | `HTTP_CLIENT_IP_FROM` |
| `http.metrics.enabled` | `HTTP_METRICS_ENABLED` |
| `http.cors.origins` | `HTTP_CORS_ORIGINS` |
| `database.host` | `DATABASE_HOST` |
| `database.password` | `DATABASE_PASSWORD` |
| `tracing.enabled` | `TRACING_ENABLED` |
| `tracing.endpoint` | `TRACING_ENDPOINT` |
| `tracing.sample_ratio` | `TRACING_SAMPLE_RATIO` |

The rule is uniform — there are no bare aliases. `APP_ENVIRONMENT` overrides
`app.environment`; a plain `ENVIRONMENT` is ignored.

For local work, [direnv](https://direnv.net/) is a convenient way to set these
per workspace.

---

## Sections

The `Config` struct is split into sections so a downstream component can be
handed only what it needs — the database service takes `DatabaseConfig`, the
HTTP server takes `HTTPConfig`, neither sees the whole object.

### `[app]` — `AppConfig`

| Key | Default | Notes |
| --- | --- | --- |
| `name` | `gopherizer` | Required. Titles the OpenAPI document and labels telemetry |
| `environment` | `local` | Required. `local`, `staging`, `production`, … |
| `version` | `1.0.0` | Required. Also the OpenAPI document version |
| `log_level` | `""` | `debug`, `info`, `warn`, `error`. Empty derives from `environment` |

These feed observability: they label `app_build_info` and the trace resource,
so a dashboard can tell which build is running and where.

### Log level

`log_level` is the only switch for verbosity. When empty it is derived:
`local` and `development` log at debug, everything else at info.

An unrecognised value **fails at startup** rather than falling back to a level
nobody chose.

Two things worth knowing:

- The `ENVIRONMENT` variable that the [slogging](https://github.com/softika/slogging)
  library reads on its own is **not** consulted here. Configuration decides the
  level, so there is one switch rather than two that can silently disagree.
- Production derives **info**, not error. slogging's own mapping uses error,
  which discards startup, shutdown and the detail behind every readiness
  failure — making production the environment you can see least about.

### `[http]` — `HTTPConfig`

| Key | Default | Notes |
| --- | --- | --- |
| `host` | `0.0.0.0` | |
| `port` | `8080` | Required |
| `read_timeout` | `2m` | |
| `read_header_timeout` | `10s` | Falls back to 10s in code if unset |
| `write_timeout` | `2m` | |
| `idle_timeout` | `2m` | |
| `request_timeout` | `30s` | Per-request deadline. `0` disables it |
| `max_body_bytes` | `1048576` | Request body cap, 1 MiB. `0` disables it |

#### Layered timeouts

Three deadlines bound a request, and the order matters: the innermost fires
first, so the error names the real cause instead of the outermost backstop
tripping on everything.

```
database.statement_timeout  15s   postgres cancels a runaway query
http.request_timeout        30s   the request's own deadline
http.write_timeout           2m   the backstop
```

A request that exceeds `request_timeout` is answered **503**, not 500 — it was
too slow, not broken, and those need different responses from a caller.

#### On `max_body_bytes`

huma already applies a 1 MiB default to any operation that reads a body, so
this is not the difference between bounded and unbounded. What it adds is an
explicit, configurable limit that also covers routes huma never sees. The same
value is passed to the operations that read a body, so raising it actually
works — otherwise huma's own default would silently overrule it.

### `[http.rate_limit]`

| Key | Default | Notes |
| --- | --- | --- |
| `requests` | `100` | Allowed per window, per client. `0` disables limiting |
| `window` | `1m` | `0` disables limiting |

### `[http.client_ip]`

Declares the trust model used to resolve a caller's address for rate limiting.
Getting this wrong either lumps every client behind a proxy into one bucket, or
lets callers spoof their identity via a header — so it is stated explicitly
rather than guessed.

| Key | Default | Notes |
| --- | --- | --- |
| `from` | `remote_addr` | `remote_addr` \| `xff` \| `header` |
| `trusted_proxies` | `0` | Number of reverse proxies in front, used with `xff` |
| `trusted_header` | `""` | Proxy-set header name, used with `header` |

If `from = "header"` but `trusted_header` is empty, the resolver falls back to
`remote_addr` — the safe, non-spoofable source.

### `[http.metrics]`

| Key | Default | Notes |
| --- | --- | --- |
| `enabled` | `true` | |
| `path` | `/metrics` | Falls back to `/metrics` if empty |

The endpoint is **unauthenticated**. Keep it off the public listener in
production, or restrict it at the ingress.

### `[http.cors]`

| Key | Default | Notes |
| --- | --- | --- |
| `origins` | `*` | Comma-separated. Empty disables CORS entirely |
| `methods` | `HEAD,GET,POST,PUT,PATCH,DELETE` | |
| `headers` | `Content-Type,Content-Length` | |

Credentials are enabled automatically **unless** `origins` contains `*`.

### `[database]` — `DatabaseConfig`

| Key | Default | Notes |
| --- | --- | --- |
| `host` | `localhost` | Required |
| `port` | `5432` | Required |
| `user` | `postgres` | Required |
| `password` | placeholder | Required. Supply the real value from the environment or a secret manager |
| `dbname` | `gopher` | Required |
| `sslmode_disabled` | `true` | Convenient locally. Enable TLS for any database that is not on localhost |
| `max_conns` | `10` | Pool ceiling |
| `min_conns` | `2` | Connections kept warm |
| `max_conn_lifetime` | `1h` | Retire connections regardless of health |
| `max_conn_idle_time` | `30m` | Release unused connections |
| `health_check_period` | `1m` | How often idle connections are verified |
| `statement_timeout` | `15s` | Applied by Postgres. `0` disables it |

State `max_conns` rather than leaving it: pgx otherwise derives the ceiling from
the host CPU count, so the same image gets a different limit on every machine
size — while the database's connection limit is a shared budget no single
service should size by accident.

`max_conn_lifetime` is what lets a failover or a rotated credential take effect
without restarting the process.

`statement_timeout` is enforced by Postgres, so a runaway query is cancelled
server-side rather than holding a pooled connection until the write deadline.
**It applies to migrations too**, since they share this configuration — raise it
for a deployment that runs long ones.

### `[tracing]` — `TracingConfig`

| Key | Default | Notes |
| --- | --- | --- |
| `enabled` | `false` | Off by default, so the template runs with no collector |
| `endpoint` | `localhost:4318` | OTLP/HTTP receiver as `host:port`, no scheme |
| `insecure` | `true` | Plain HTTP. Fine for a collector on localhost, wrong across a network |
| `sample_ratio` | `1.0` | Head sampling for traces started here, `0`–`1` |

Export is OTLP over **HTTP**, not gRPC: the gRPC exporter pulls in
`google.golang.org/grpc`, which is a large dependency for a template to impose.
Traces go straight to a backend that accepts OTLP — no collector is required in
between, though one can be pointed at instead.

`sample_ratio` only governs traces that *start* here. A request arriving with a
`traceparent` header follows the caller's sampling decision, so a trace stays
whole across services instead of developing holes wherever a hop re-rolled.

Turning `enabled` off removes the tracing middleware entirely. Correlation ids
are unaffected, which is why they are kept alongside trace ids rather than
replaced by them.

---

## Adding a new setting

1. Add the key to `default.toml`, under an existing section or a new one.
2. Add the field to the matching struct in `config.go` with a `mapstructure`
   tag, and a `validate:"required"` tag if the service cannot start without it.
3. Pass only the relevant section to the component that needs it.

---

## Best practices

1. **Keep secrets out of version control.**
   `default.toml` holds development placeholders only. Real credentials belong
   in environment variables or a secret manager.

2. **Validate early.**
   `config.New()` validates at startup so a misconfiguration surfaces
   immediately, not on the first request that happens to need the value.

3. **Review the permissive defaults before deploying.**
   Wildcard CORS, `remote_addr` client-IP resolution, an open metrics endpoint
   and disabled database TLS are all fine locally and all wrong in production.
   [SECURITY.md](../SECURITY.md) lists them with their consequences.
