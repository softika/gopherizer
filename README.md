![go workflow](https://github.com/softika/gopherizer/actions/workflows/test.yml/badge.svg)
![lint workflow](https://github.com/softika/gopherizer/actions/workflows/lint.yml/badge.svg)

# Project gopherizer

This is a golang template repository. A good place to start your project.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## Project Structure

Top Level Directories

- [cmd/](cmd) - cli commands like `migrate` and `server`.
- [config/](config) - configuration and loading environment variables.
- [database/](database) - database service, repositories and migration files.
- [http/](http) - deals with transport layer, input sanitation and consumes `internal/services` layer.
- [internal/](internal) - core logic, `services` as business use cases and `model` as domain entities.
- [logging/](logging) - structural logging service.
- [testinfra/](testinfra) - test-containers utilities.
- [tests/](tests) - e2e tests.

### http
`http` folder deals with the transport protocol and typically input sanitation.
For example, there could be:
- `http/api` for a REST API implementation
- `http/grpc` for a gRPC server implementation
- `http/server` for running the server

Ideally they all should only consume `internal/services` layer and do not deal
with business logic directly.

### internal
`internal` contains all reusable business logic that can be consumed by `http` or `eventing` layers.
It is divided into:
- `services` - represents business logic use cases.
- `model` - performs business logic and deals directly with `storage` layer and/or `integrations` clients.

### Environment Config

All required environment variables to run this service is defined on `config/default.config`.
When you run `make run`, it will check if `config/config` exists and will create it from
`config/default.config` if it does not exist.

The value on this file can be overridden by setting the equivalent environment variable.
For example, to change the:

- `environment` value you can set `ENVIRONMENT`
- `http.host` value you can set `HTTP_HOST`
  on your machine to match your desired value.

You may also use [direnv](https://direnv.net/) to define environment variable on a workspace basis.

#### Environment Struct

Environment Struct is split into sections for easier readability and passing of configs to downstream services. For
example, the database service would only require the `DatabaseConfig` instead of the full config object.

The structure all belongs in `Config` which holds all the configs used in the service. Individual configs are defined
inside `Config` as a struct. This allows for individually passing structs of a specific config to downstream services.

example:

```go
package example 

import (
    "context"
    "database/sql"
    "fmt"

    // pgx 
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jackc/pgx/v5/stdlib"
	
    "github.com/softika/gopherizer/config"
)

type Service struct {
  db   *sql.DB
  pool *pgxpool.Pool
}

func New(cfg config.DatabaseConfig) Service {
    dsn := fmt.Sprintf(
        "postgresql://%s:%s@%s:%s/%s?sslmode=require",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, 
    )
	
    pool, err := pgxpool.New(context.Background(), dsn)
    if err != nil {
        panic(err)
    }

    db := stdlib.OpenDBFromPool(pool)
    if err = db.Ping(); err != nil {
        panic(err)
    }

    return Service{
        db:   db,
        pool: pool,
    }
}
```
##### AppConfig

AppConfig is basic config of the application such as the name, environment and version. These are usually used for
observability such that we can discern the versions of the service and the environment it is running in.

### Adding new migration file

We use [goose](https://github.com/pressly/goose) to run
SQL database migration and managing migration files.

To create a new migration file.
```sh
goose -dir database/migrations create xxx sql
```

### Generating mocks

We use [gomock](https://github.com/uber-go/mock) to generate mocks.

If you change the interface and need to create/update the generated mock, make
sure to always run this command.

```sh
# note that PWD should be the root package of the source generation files.
make mocks
```

## Testing

To run the tests locally, run `make test` to run all the unit tests
or run `go ./... -run <test-name>` to run specific unit test.

By default `make test` will run the tests in parallel n-times.
You can also do this manually by running `go test ./... -parallel -count=5`



## MakeFile
Check the [Makefile](Makefile) for more available commands.