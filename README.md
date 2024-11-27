![go workflow](https://github.com/softika/gopherizer/actions/workflows/test.yml/badge.svg)
![lint workflow](https://github.com/softika/gopherizer/actions/workflows/lint.yml/badge.svg)

# Gopherizer

The motivation behind creating this template repository was to establish a unified architecture across multiple repositories, eliminating the need to repeat boilerplate code. This approach not only ensures consistency and maintainability but also provides the flexibility to extend and adapt the architecture as needed. By leveraging this template, teams can focus on developing unique features rather than reinventing the wheel for each project.

## Features
- ✅ HTTP Server run with graceful shutdown
- ✅ Routing with [Chi](https://go-chi.io/#/README) - easy to swap with other routers
- ✅ Database Service (Postgres)
- ✅ Migrations ([goose](https://github.com/pressly/goose))
- ✅ Dynamic configuration 
- ✅ Structured [logging](https://github.com/softika/slogging) 
- ✅ Centralized error Handling 
- ✅ Integration testing with [Testcontainers](https://golang.testcontainers.org/)
- ✅ CI Pipeline (GitHub Actions)
- ✅ Dockerized development environment
- ✅ OpenAPI Documentation
- 🏗️ OpenTelemetry


## Project Structure

- [api/](api) - http server, handlers and routes.
- [cmd/](cmd) - cli commands, `serve` and `migrate`.
- [config/](config) - configuration and loading environment variables.
- [database/](database) - database service, repositories and migration files.
- [internal/](internal) - core logic, `services` as business use cases and `model` as domain entities.
- [pkg/](pkg) - reusable packages.
- [tests/](tests) - e2e tests.

### Building and running your application
When you're ready, start your application by running:

``` bash
make compose
``` 

Your application will be available at http://localhost:8080.

### Deploying your application to the cloud

First, build your image, e.g.: `docker build -t myapp .`.
If your cloud uses a different CPU architecture than your development
machine (e.g., you are on a Mac M1 and your cloud provider is amd64),
you'll want to build the image for that platform, e.g.:
`docker build --platform=linux/amd64 -t myapp .`.

Then, push it to your registry, e.g. `docker push myregistry.com/myapp`.

Consult Docker's [getting started](https://docs.docker.com/go/get-started-sharing/)
docs for more detail on building and pushing.

#### References
* [Docker's Go guide](https://docs.docker.com/language/golang/)

### Environment Config

All required environment variables for running this service are defined in `config/default.ini`.

Values in this file can be overridden by setting the corresponding environment variables. For example:

- Set `ENVIRONMENT` to change the `environment` value
- Set `HTTP_HOST` to adjust the `http.host` value to your desired setting

Additionally, you can use [direnv](https://direnv.net/) to define environment variables on a per-workspace basis.

#### Environment Struct

The `Config` struct is organized into sections to improve readability and make it easier to pass specific configurations to downstream services. 
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

Check the [Makefile](Makefile) for more available commands.</br>
Run `make help` to see all available commands.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

If you have any suggestions, questions or want to contribute, feel free to create an issue or a pull request.