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
- ✅ Centralized error handling — domain error types mapped to status codes, detail kept server-side
- ✅ Liveness and readiness probes, separated
- ✅ Prometheus metrics with bounded label cardinality
- ✅ Per-client rate limiting with an explicit client-IP trust model
- ✅ Configurable CORS
- ✅ Integration and e2e testing with [Testcontainers](https://golang.testcontainers.org/)
- ✅ CI pipeline (GitHub Actions): build, test, `golangci-lint` v2, `govulncheck`, Dependabot
- ✅ Dockerized development environment, non-root runtime image
- 🏗️ OpenTelemetry tracing

Authentication and authorization are deliberately **not** included — see
[what is not included](#what-is-not-included).

## OpenAPI

The OpenAPI document is generated from the registered operations, so it cannot
drift from the routes the server actually serves. There is no spec file to
hand-maintain. See [api/README.md](api/README.md).

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
- [pkg/](pkg) - reusable packages: `errorx` domain errors, `testinfra` test containers.
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
- `HTTP_PORT` overrides `http.port`
- `HTTP_CLIENT_IP_FROM` overrides `http.client_ip.from`
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
func New(cfg config.DatabaseConfig) (Service, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    pool, err := pgxpool.New(ctx, dsnFromConfig(cfg))
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

`AppConfig` provides essential application settings: name, environment and
version. These feed observability — they identify the service version and the
environment it runs in, and the name and version also title the generated
OpenAPI document.

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

The template ships with **no authentication or authorization**. Adding them is
the first thing to do before exposing a service publicly. Distributed tracing
is also absent — the exporter, backend, sampling strategy and propagation
format are deployment decisions, so no choice is baked in here.

See [SECURITY.md](SECURITY.md) for the full list of what this template is and
is not responsible for, and how to report a vulnerability.

## MakeFile

Check the [Makefile](Makefile) for more available commands.</br>
Run `make help` to see all available commands.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

If you have any suggestions, questions or want to contribute, feel free to create an issue or a pull request.
