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
| `http.port` | `HTTP_PORT` |
| `http.read_header_timeout` | `HTTP_READ_HEADER_TIMEOUT` |
| `http.rate_limit.requests` | `HTTP_RATE_LIMIT_REQUESTS` |
| `http.client_ip.from` | `HTTP_CLIENT_IP_FROM` |
| `http.metrics.enabled` | `HTTP_METRICS_ENABLED` |
| `http.cors.origins` | `HTTP_CORS_ORIGINS` |
| `database.host` | `DATABASE_HOST` |
| `database.password` | `DATABASE_PASSWORD` |

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

| Key | Notes |
| --- | --- |
| `name` | Required. Also titles the generated OpenAPI document |
| `environment` | Required. `local`, `staging`, `production`, … |
| `version` | Required. Also the OpenAPI document version |

These feed observability: they identify which build is running, and where.

### `[http]` — `HTTPConfig`

| Key | Default | Notes |
| --- | --- | --- |
| `host` | `0.0.0.0` | |
| `port` | `8080` | Required |
| `read_timeout` | `2m` | |
| `read_header_timeout` | `10s` | Falls back to 10s in code if unset |
| `write_timeout` | `2m` | |
| `idle_timeout` | `2m` | |

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
