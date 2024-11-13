![go workflow](https://github.com/softika/gopherizer/actions/workflows/test.yml/badge.svg)
![lint workflow](https://github.com/softika/gopherizer/actions/workflows/lint.yml/badge.svg)

# Gopherizer

This is a Go template repository, providing a solid foundation for starting new projects.

## Features
- [x] HTTP Server run with graceful shutdown
- [x] JWT Authentication
- [x] Database Service (Postgres)
- [x] Migrations (goose)
- [x] Configuration 
- [x] Logging 
- [x] Error Handling 
- [x] Testing
- [x] CI Pipeline (Github Actions)
- [ ] OpenAPI Documentation
- [ ] Google Auth


## Project Structure

Top Level Directories

- [api/](api) - http server, handlers and routes.
- [cmd/](cmd) - cli commands like `migrate` and `server`.
- [config/](config) - configuration and loading environment variables.
- [database/](database) - database service, repositories and migration files.
- [internal/](internal) - core logic, `services` as business use cases and `model` as domain entities.
- [pkg/](pkg) - reusable packages.
- [tests/](tests) - e2e tests.

### Environment Config

All required environment variables for running this service are defined in `config/default.config`. 
When you run `make run`, it checks for `config/config` and will create it from `config/default.config` 
if it doesn't already exist.

Values in this file can be overridden by setting the corresponding environment variables. For example:

- Set `ENVIRONMENT` to change the `environment` value
- Set `HTTP_HOST` to adjust the `http.host` value to your desired setting

Additionally, you can use [direnv](https://direnv.net/) to define environment variables on a per-workspace basis.

#### Environment Struct

The `Environment` struct is organized into sections to improve readability and make it easier to pass specific configurations to downstream services. 
For instance, the database service only needs `DatabaseConfig` rather than the entire configuration object.

All configuration sections are contained within the `Config` struct, which holds every configuration used in the service. 
Each individual configuration is defined as a struct within `Config`, enabling selective passing of specific configurations to downstream services.

example:

```go
package database 

import (
    "context"
    "fmt"

    // pgx 
    "github.com/jackc/pgx/v5/pgxpool"
	
    "github.com/softika/gopherizer/config"
)

type Service struct {
    pool *pgxpool.Pool
}

func New(cfg config.DatabaseConfig) Service {
    dsn := fmt.Sprintf(
        "postgresql://%s:%s@%s:%s/%s?sslmode=require",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, 
    )

    ctx := context.Background()
	
    pool, err := pgxpool.New(ctx, dsn)
    if err != nil {
        panic(err)
    }

	if err = pool.Ping(ctx); err != nil {
        panic(err)
    }

    return Service{
        pool: pool,
    }
}
```
##### AppConfig

AppConfig provides essential application settings, including the name, environment, and version. 
These settings are typically used for observability, 
allowing us to identify the service version and the environment in which it is running.

### Adding new migration file

We use [goose](https://github.com/pressly/goose) to run
SQL database migration and managing migration files.

To create a new migration file.
```sh
goose -dir database/migrations create xxx sql
```

### Generating mocks

We use [gomock](https://github.com/uber-go/mock) to generate mocks.

If you change the interface make sure to always run this command:
```sh
make mocks
```

## Testing

To run the tests locally, run `make test` to run all the unit tests
or run `go ./... -run <test-name>` to run specific unit test.

By default `make test` will run the tests in parallel n-times.
You can also do this manually by running: `go test ./... -parallel -count=5`


## MakeFile
Check the [Makefile](Makefile) for more available commands.