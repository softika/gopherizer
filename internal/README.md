# Internal package documentation

The `internal` package holds the core functionality and business logic. Go
enforces the boundary: nothing outside this module can import it, which keeps
the domain logic from becoming an accidental public API.

## Overview

Each domain lives in its own subpackage and follows the same shape:

```
internal/<domain>/
├── model.go      domain entity
├── request.go    inbound types, with validation constraints
├── response.go   outbound types, and the model → response mapping
├── service.go    business use cases
└── mock/         generated mocks for the repository interface
```

Two domains ship as examples: [`profile/`](profile) (full CRUD over a database)
and [`health/`](health) (liveness and readiness). Neither is load-bearing —
replace or delete them to suit your application.

## Shared building blocks (`base.go`)

| Type | Purpose |
| --- | --- |
| `Base` | Embeddable `Id` / `CreatedAt` / `UpdatedAt` for entities |
| `Repository[T, ID]` | Generic CRUD interface that concrete repositories satisfy |
| `PageRequest`, `Page[T]` | Pagination scaffolding — declared, not yet wired to any endpoint |

## The layers

### Model

A plain struct embedding `internal.Base`, with `db` tags describing the column
mapping.

### Request and response types

Requests carry **JSON Schema validation tags**, which
[huma](https://huma.rocks) both enforces on every request and uses to generate
the OpenAPI schema — one declaration, no drift:

```go
type CreateRequest struct {
    FirstName string `json:"firstName" minLength:"1" maxLength:"72" doc:"Given name" example:"John"`
    LastName  string `json:"lastName" minLength:"1" maxLength:"72" doc:"Family name" example:"Doe"`
}
```

Note that JSON Schema `required` only asserts **presence**, so a non-empty
string needs `minLength`. Without it, `{"firstName": ""}` is accepted.

Responses are separate types with explicit `json` tags and a `fromModel` mapper.
Keeping them distinct from the model means a column added to the entity does not
silently appear in the API.

### Service

The use-case layer. It depends on a repository **interface** declared in the
same package, never on a concrete implementation — which is what allows the
database layer to be swapped or mocked.

Services log the full failure with context and re-wrap it with `errorx`,
preserving the type the repository assigned:

```go
func (s Service) GetById(ctx context.Context, req GetRequest) (*Response, error) {
    u, err := s.repo.GetById(ctx, req.Id)
    if err != nil {
        slog.ErrorContext(ctx, "failed to get profile by id", "id", req.Id, "error", err)
        return nil, errorx.NewError(
            fmt.Errorf("failed to get profile by id: %w", err),
            errorx.TypeOf(err),
        )
    }

    res := new(Response)
    return res.fromModel(u), nil
}
```

`slog.ErrorContext` is used rather than `slog.Error` so the values on the
context reach the log record: the request and correlation ids always, and the
trace and span ids whenever the request was sampled. Passing the context is the
whole mechanism — `slog.Error` would drop all of them. The API layer maps the
`errorx` type to a status code and replaces the message with a safe one — see
[api/README.md](../api/README.md#error-handling-errorsgo).

### Mocks

Repository interfaces are marked with a `go:generate` directive:

```go
//go:generate mockgen -source=service.go -destination=./mock/service.go -package=mock
```

Run `make mocks` after changing an interface.

---

## Health as a worked example

The `health` package shows why the layering is worth the ceremony. It splits
one endpoint into two:

- **`Live`** deliberately touches no dependency. Liveness answers "should this
  process be restarted"; checking the database here would let a brief outage
  cause an orchestrator to kill healthy pods, turning a recoverable blip into a
  restart storm.
- **`Check`** reports readiness and returns `errorx.ErrUnavailable` when a
  dependency is down, so the caller sees a 503 and load balancers stop routing
  here — instead of a 200 that hides the outage.

Its `Response` is deliberately minimal (`{"status":"up"}`). Connection counts
and timings describe the infrastructure, so they are logged for operators rather
than served to callers.

---

## Adding a domain

1. Create `internal/<domain>/` with the files above.
2. Declare the repository interface in the service package; implement it under
   [`database/repositories/`](../database/repositories).
3. Register the repository and service in
   [`api/bootstrap.go`](../api/bootstrap.go).
4. Register the endpoints in [`api/operations.go`](../api/operations.go).

## Best practices

- **Keep internal logic encapsulated.** Nothing here should need to be
  importable from outside the module.
- **Depend on interfaces, not implementations.** The service declares what it
  needs; the database package provides it.
- **Return typed errors.** `errorx` is how the HTTP layer learns whether a
  failure is a 404, a 503 or a 500, without knowing anything about SQL.
- **Keep responses separate from models.** They change for different reasons.
