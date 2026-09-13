![build workflow](https://github.com/softika/gopherizer/actions/workflows/test.yml/badge.svg)
![lint workflow](https://github.com/softika/gopherizer/actions/workflows/lint.yml/badge.svg)
![security workflow](https://github.com/softika/gopherizer/actions/workflows/security.yml/badge.svg)

# Gopherizer

The motivation behind creating this template repository was to establish a unified architecture across multiple repositories, eliminating the need to repeat boilerplate code. This approach not only ensures consistency and maintainability but also provides the flexibility to extend and adapt the architecture as needed. By leveraging this template, teams can focus on developing unique features rather than reinventing the wheel for each project.

## Requirements

| | |
| --- | --- |
| Go | 1.27 or newer |
| PostgreSQL | 18 or newer — the `profiles` primary key defaults to `uuidv7()`, which landed in 18 |
| Docker | required for `make start` and for the testcontainers-backed test suites |

## Features

- ✅ HTTP server with graceful shutdown on `SIGINT` and `SIGTERM`
- ✅ Routing with [Chi](https://go-chi.io/#/README) — easy to swap with any `http.Handler`
- ✅ Request decoding, validation and OpenAPI generated from Go types with [huma](https://huma.rocks)
- ✅ Database service (Postgres) on [pgx](https://github.com/jackc/pgx) pooling
- ✅ Migrations ([goose](https://github.com/pressly/goose)), embedded in the binary
- ✅ Dynamic configuration, overridable by environment variables
- ✅ Structured [logging](https://github.com/softika/slogging) with request and correlation ids
- ✅ Structured access log, on the same stream and format as application logs
- ✅ Centralized error handling — domain error types mapped to status codes, detail kept server-side
- ✅ Liveness and readiness probes, separated
- ✅ Prometheus metrics with bounded label cardinality, including connection pool saturation
- ✅ OpenTelemetry tracing over OTLP/HTTP, covering HTTP requests and every database query
- ✅ Exemplars linking a latency histogram to the trace that produced it
- ✅ Layered timeouts — Postgres statement, per-request deadline, server write
- ✅ Explicit connection pool sizing, not inherited from the host CPU count
- ✅ Request body cap, applied uniformly across every route
- ✅ Panics reported as structured logs with correlation and trace ids
- ✅ OIDC bearer token verification, with per-operation scope and role checks
- ✅ Per-client rate limiting with an explicit client-IP trust model
- ✅ Configurable CORS
- ✅ Integration and e2e testing with [Testcontainers](https://golang.testcontainers.org/)
- ✅ CI pipeline (GitHub Actions): build, test, `golangci-lint` v2, `govulncheck`, Dependabot
- ✅ Dockerized development environment, non-root runtime image
- ✅ Local observability stack (Prometheus, Tempo, Grafana) behind a compose profile

Authentication is **off by default** and turns on with configuration — see
[Authentication](#authentication). Token *issuance* is deliberately not
included; see [what is not included](#what-is-not-included).

## OpenAPI

The OpenAPI document is generated from the registered operations, so it cannot
drift from the routes the server actually serves. There is no spec file to
hand-maintain. See [api/README.md](api/README.md).

## Observability

Three signals, and they are joined rather than merely present.

| Signal | Where | Notes |
| --- | --- | --- |
| Logs | stdout, JSON | Every record carries `X-Request-Id` and `X-Correlation-Id`; sampled requests also carry `trace_id` and `span_id` |
| Metrics | `GET /metrics` | Request rate and latency by route pattern, connection pool statistics, `app_build_info` |
| Traces | OTLP/HTTP | A server span per request and a client span per database query |

**Correlation ids and trace ids both exist, on purpose.** Trace ids only appear
on sampled requests, and tracing can be switched off entirely; correlation ids
are present on every request either way, and a system that cannot speak W3C
`traceparent` can still echo a header. Losing request correlation because a
request happened not to be sampled would be the worst kind of gap — an
intermittent one.

The three connect through **exemplars**: the latency histogram carries the
`trace_id` of sampled requests, so a spike on a Grafana panel is one click from
the trace that caused it, and that trace's ids lead back to the logs for the
same request.

Tracing is **off by default**, so nothing here requires a collector to be
running. To start the full local stack:

```bash
make observability
```

| | |
| --- | --- |
| Grafana | http://localhost:3000 — **start here.** Prometheus and Tempo pre-provisioned and linked |
| Prometheus | http://localhost:9090 — exemplar storage enabled |
| Tempo | `localhost:3200` — API only, **no web UI** |

Two dashboards are provisioned from disk at startup, under the **Gopherizer**
folder in Grafana:

| Dashboard | Shows |
| --- | --- |
| Service overview | Request rate, server error ratio, latency percentiles by route, and the running build. The latency panel carries exemplars — click a diamond to open that request's trace |
| Database pool | Connections by state against the pool maximum, acquire waits, and connection churn by reason |

They live in [`deploy/observability/grafana/dashboards/`](deploy/observability/grafana/dashboards)
as plain JSON. Editing them in the Grafana UI works, but the files are the
source of truth and overwrite UI changes on the next scan — copy anything worth
keeping back into the JSON.

Tempo serves no browsable page: `http://localhost:3200/` returns 404 because
`/` is not a route it has. Traces are read in Grafana, which queries Tempo
through the provisioned datasource. The port is exposed so Grafana can reach it
and so the API can be queried directly:

```sh
curl -s localhost:3200/ready                  # readiness
curl -s "localhost:3200/api/search?limit=5"   # recent traces
curl -s localhost:3200/api/traces/<trace_id>  # one trace, by id
```

Prometheus scrapes both `server:8080` and `host.docker.internal:8080`, so it
finds the application whether it runs in the compose stack or on the host via
`make run`. Whichever is not in use shows as a down target; that is expected.

Query arguments are never attached to a database span. Statements are
parameterised, so the recorded text carries no values, while the arguments hold
real ones — and spans leave the process for a third-party backend.

## Authentication

The service is an OAuth2 **resource server**: it validates bearer tokens issued
by an identity provider and issues none of its own. There is no login flow, no
callback route and no session — see [what is not included](#what-is-not-included).

It is **off by default**, so the template runs and its tests pass with no
identity provider anywhere. Turning it on takes configuration and nothing else:

```bash
OIDC_ENABLED=true \
OIDC_ISSUER=https://id.example.com/realms/myapp \
OIDC_AUDIENCE=myapp-api \
make run
```

At startup the service reads the provider's discovery document, caches its key
set, and from then on verifies every token's signature, issuer, audience and
expiry locally — no call to the provider per request.

### Try it against a real provider

```bash
make auth   # starts the stack with Keycloak, realm pre-configured
```

Then fetch a token and use it:

```bash
TOKEN=$(curl -s -d grant_type=password -d client_id=gopherizer-cli \
  -d username=gopher -d password=gopher \
  http://localhost:8081/realms/gopherizer/protocol/openid-connect/token | jq -r .access_token)

curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/profile/{id}
```

The realm ships two users: `gopher` holds the `admin` role, `reader` does not.
Both can read and write; only `gopher` can delete, which is what demonstrates
the role check. `make auth-host` runs Keycloak for an application started on the
host with `make run` instead — see the note on issuer addresses below.

### Declaring what an operation requires

A requirement is declared once, at registration, and is both enforced and
documented from that single value:

```go
huma.Register(api, secured(huma.Operation{
    OperationID: "create-profile",
    Method:      http.MethodPost,
    Path:        "/api/v1/profile",
}, authx.Scope("profile:write"), g), handler)
```

`authx` composes requirements with `Scope`, `Role`, `AllOf` and `AnyOf`. A
handler reads the caller with `authx.FromContext(ctx)`, which is how a check
the middleware cannot make — "does this subject own this record?" — is written
in the service layer.

Scopes reach the generated OpenAPI document; **roles cannot**, because an
OpenAPI security requirement has no way to express them. The projection runs one
way, from the requirement that is enforced, so the document can only ever
understate what is enforced, never overstate it.

### What stays open

The health probes, `/metrics`, `/docs` and `/openapi.json` are never guarded.
An orchestrator carries no token, and a probe that answered 401 would take every
instance out of rotation for an identity provider outage the fleet would
otherwise have served straight through.

### Things that bite

| | |
| --- | --- |
| `oidc.audience` must be the **API's** identifier, never an OAuth client id | An ID token's `aud` *is* the client id, so reusing one lets a browser-held ID token pass as an access token. Tokens carrying `at_hash`, `c_hash` or `nonce` are refused for the same reason. |
| The issuer must match `iss` **byte for byte** | Including the trailing slash. A provider reached on two addresses stamps whichever one it was configured with — which is why the Keycloak profile pins `KC_HOSTNAME` and why `make auth-host` exists. |
| Roles are usually nested | Keycloak puts them at `realm_access.roles`, Entra and Auth0 at `roles`. `OIDC_ROLES_CLAIM` takes a dotted path; the shipped default is the vendor-neutral `roles`. |
| Startup fails if the provider is unreachable | Deliberate. A process that starts unable to verify anything is not degraded — it reports itself healthy, accepts traffic, and rejects every request. A running process is unaffected: keys are cached. |

There is no setting for skipping the audience, issuer, expiry or signature
check, and the accepted signing algorithms are pinned in code. Both omissions
are deliberate: each would be one environment variable away from turning
authentication off in a deployment.

## Building and running your application

Start the whole stack — database, migrations, server:

```bash
make start
```

Your application will be available at http://localhost:8080, with the API
reference at http://localhost:8080/docs.

To run the server on the host against a containerized database instead:

```bash
make db-start
make migrate-up
make run
```

Run `make help` to see all available commands.

## Project structure

- [api/](api) - http server, router, middleware and huma operations. More about api [here](api/README.md).
- [cmd/](cmd) - cli commands, `serve` and `migrate`.
- [config/](config) - configuration and environment variable loading. More about config [here](config/README.md).
- [database/](database) - database service, transactions, repositories and migration files. More about database [here](database/README.md).
- [internal/](internal) - core logic, `services` as business use cases and `model` as domain entities. More about internal [here](internal/README.md).
- [pkg/](pkg) - reusable packages: `errorx` domain errors, `logx` logger construction, `otelx` tracing setup, `authx` identity and authorization rules, `oidcx` token verification, `testinfra` test containers.
- [deploy/](deploy) - configuration for the local observability stack.
- [tests/](tests) - e2e tests.

## Deploying your application to the cloud

First, build your image, e.g.: `docker build -t myapp .`.
If your cloud uses a different CPU architecture than your development
machine (e.g., you are on a Mac M1 and your cloud provider is amd64),
you'll want to build the image for that platform, e.g.:
`docker build --platform=linux/amd64 -t myapp .`.

Then, push it to your registry, e.g. `docker push myregistry.com/myapp`.

The runtime image is Alpine, pinned to a minor version, and runs as a
non-privileged user. Consult Docker's
[getting started](https://docs.docker.com/go/get-started-sharing/) docs for more
detail on building and pushing.

#### References
* [Docker's Go guide](https://docs.docker.com/language/golang/)

## Configuration

All settings for running this service are defined in
[`config/default.toml`](config/default.toml), which is embedded in the binary.
Any value can be overridden by an environment variable: uppercase the key and
replace `.` with `_`.

- `APP_ENVIRONMENT` overrides `app.environment`
- `APP_LOG_LEVEL` overrides `app.log_level`
- `HTTP_PORT` overrides `http.port`
- `HTTP_CLIENT_IP_FROM` overrides `http.client_ip.from`
- `TRACING_ENABLED` overrides `tracing.enabled`
- `OIDC_ENABLED` overrides `oidc.enabled`
- `OIDC_ISSUER` overrides `oidc.issuer`
- `OIDC_AUDIENCE` overrides `oidc.audience`
- `DATABASE_PASSWORD` overrides `database.password`

Additionally, you can use [direnv](https://direnv.net/) to define environment
variables on a per-workspace basis.

Several defaults are permissive for local development and must be reviewed
before deploying — CORS origins, the client-IP trust model, the metrics
endpoint and database TLS. They are listed with their consequences in
[SECURITY.md](SECURITY.md). Full details in [config/README.md](config/README.md).

### Config struct

The `Config` struct is organized into sections to improve readability and make
it easier to pass specific configurations to downstream services. For instance,
the database service takes only `DatabaseConfig` rather than the entire
configuration object:

```go
package database

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/softika/gopherizer/config"
)

// New opens a connection pool and verifies it is reachable. The caller owns the
// returned Service and must Close it, which is what allows a clean shutdown.
// Options adjust the pool before it opens; WithQueryTracer attaches tracing.
func New(cfg config.DatabaseConfig, opts ...Option) (Service, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    poolCfg, err := pgxpool.ParseConfig(dsnFromConfig(cfg))
    if err != nil {
        return nil, fmt.Errorf("failed to parse db connection config: %w", err)
    }

    for _, opt := range opts {
        opt(poolCfg)
    }

    pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create db connection pool: %w", err)
    }

    if err = pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("failed to ping db: %w", err)
    }

    return &service{pool: pool}, nil
}
```

Note that failures are returned rather than panicked, so the server reports a
clear startup error instead of a stack trace.

#### AppConfig

`AppConfig` provides essential application settings: name, environment, version
and log level. These feed observability — they label `app_build_info` and the
trace resource, and the name and version also title the generated OpenAPI
document.

`log_level` is the single switch for verbosity. Leave it empty to derive the
level from `environment`. Note that the `ENVIRONMENT` variable read by the
logging library is *not* consulted: configuration decides, so there is one
switch rather than two that can disagree.

## Database migrations

We use [goose](https://github.com/pressly/goose) to run SQL migrations and
manage migration files. They are embedded into the binary, so a deployed image
carries its own schema history.

Create a new migration file:

```sh
goose -dir database/migrations create xxx sql
```

Apply and roll back:

```sh
make migrate-up    # applies every pending migration
make migrate-down  # rolls back the most recent migration only
```

## Generating mocks

We use [gomock](https://github.com/uber-go/mock) to generate mocks. Interfaces
are marked with a `//go:generate mockgen` directive.

If you change an interface, run:

```sh
make mocks
```

## Testing

Run the full suite:

```sh
make test
```

That runs `go test -vet=off -count=1 -race -timeout=30s ./...` — every package,
with the race detector on.

The repository and e2e suites start a real PostgreSQL container through
[Testcontainers](https://golang.testcontainers.org/), so Docker must be running.
To skip them and run only the unit tests:

```sh
go test -short ./...
```

To run a single test:

```sh
go test ./... -run <test-name>
```

## What is not included

The template validates tokens; it does not **issue** them. There is no login
flow, no authorization-code or PKCE handling, no token endpoint, no refresh, no
user store and no consent screen — those belong to an identity provider, and
[Authentication](#authentication) covers pointing this service at one.

Authorization is a vocabulary, not a policy: `authx` gives you scopes, roles and
the combinators to express a rule, and the scopes the example endpoints require
are illustrative. Replace them with whatever your provider actually issues.

Tracing is included but **disabled by default**, and it exports over OTLP to
whatever backend a deployment points it at. The sampling ratio and endpoint are
deployment decisions; no vendor choice is baked in.

See [SECURITY.md](SECURITY.md) for the full list of what this template is and
is not responsible for, and how to report a vulnerability.

## MakeFile

Check the [Makefile](Makefile) for more available commands.</br>
Run `make help` to see all available commands.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

If you have any suggestions, questions or want to contribute, feel free to create an issue or a pull request.
