# Database package documentation

The `database` package provides everything needed to talk to Postgres:
connection pooling, transaction handling, repositories, and an embedded set of
SQL migrations.

It is the **only** layer that knows about the driver. Everything above it works
with domain types and `errorx` error types instead of `pgx` sentinels.

---

## Components

### Database service (`database.go`)

Manages the connection pool via
[pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool).

```go
db, err := database.New(cfg.Database)
if err != nil {
    return err
}
defer db.Close()
```

- **Each call returns an independent `Service`.** There is no package-level
  singleton, so tests can run against isolated pools and the caller owns the
  lifetime.
- **Failures are returned, not panicked**, so the server reports a clear
  startup error.
- **Connection is bounded by a 10s timeout**, so an unreachable database fails
  fast instead of hanging the process.
- **Options adjust the pool before it opens.** `WithQueryTracer` is how tracing
  is attached, without any repository knowing about it.
- `Close` must be called. `cmd/serve` closes it after the HTTP server has
  drained, and does so even when the shutdown itself failed, so a shutdown
  error cannot leak connections.

The `Service` interface exposes:

| Method | Purpose |
| --- | --- |
| `Health(ctx)` | Pool statistics and reachability, used by the readiness probe |
| `Pool()` | The `*pgxpool.Pool`, for queries |
| `DB()` | A `*sql.DB` view of the same pool, for goose |
| `Close()` | Releases the pool |

`Health` reports connection counts, acquire timings and destroy counters, and
annotates the result when the pool looks saturated. That detail is logged for
operators — the readiness endpoint returns only `up`.

### Transactions (`tx.go`)

`TxManager` abstracts begin/commit/rollback:

| Method | Purpose |
| --- | --- |
| `Begin(ctx)` | Starts a transaction |
| `Execute(ctx, fn)` | Runs `fn` in a transaction, committing when it returns nil and rolling back otherwise |

Two details are load-bearing:

- **The result is named.** The deferred function decides between commit and
  rollback; an unnamed result would be copied out before that runs, hiding a
  failed `COMMIT` from the caller.
- **A rollback error never replaces the original.** The business failure is
  what the caller needs; a rollback error is joined onto it as additional
  context via `errors.Join`.

Panics are rolled back and re-thrown, preserving the original stack. A rollback
of an already-closed transaction is treated as benign.

```go
type ProfileRepository struct {
    database.TxManager
    database.Service
}

func (r ProfileRepository) UpdateWithTx(ctx context.Context, p *profile.Profile) (*profile.Profile, error) {
    // add more db operations here, e.g. a SELECT ... FOR UPDATE lock by id

    err := r.Execute(ctx, func(tx pgx.Tx) error {
        return tx.QueryRow(ctx, profileUpdateSql,
            p.FirstName, // $1
            p.LastName,  // $2
            p.Id,        // $3
        ).Scan(&p.Id, &p.CreatedAt, &p.UpdatedAt)
    })

    return p, err
}
```

### Repositories (`repositories/`)

Encapsulate data access, keeping it out of the business logic. Each repository
takes a `database.Service` and gets a `TxManager` for free.

SQL lives in `repositories/sql/*.sql` and is pulled in with `//go:embed`, so
statements stay readable and syntax-highlighted rather than buried in string
literals. All queries are parameterized.

```go
var (
    //go:embed sql/profile_get_by_id.sql
    profileGetByIdSql string
)
```

Repositories satisfy the generic `internal.Repository[T, ID]` interface, so the
service layer depends on the shape, not the implementation.

### Query tracing (`tracing.go`)

`WithQueryTracer` installs a `pgx.QueryTracer` on the pool, so every query gets
a client span:

```go
db, err := database.New(cfg.Database, database.WithQueryTracer(otel.GetTracerProvider()))
```

Instrumenting at the driver is what keeps this invisible to the layers above.
No repository is annotated, no call site changes, and queries issued inside a
transaction are covered too — by construction rather than by remembering.

Two decisions are load-bearing:

- **Arguments are never recorded.** The statement text goes on the span, and
  because statements are parameterised it carries no values. `data.Args` holds
  the real ones — names, emails, tokens — and spans leave the process for a
  third-party backend. There is a test asserting no argument reaches an
  attribute.
- **`pgx.ErrNoRows` is not an error.** It is how a lookup reports "not found",
  which is an ordinary outcome. Marking those spans failed would bury the real
  failures.

#### Error classification (`repositories/errors.go`)

The repository is the only layer that can tell a missing row apart from a
genuine failure, so it is the layer that tags errors:

| Driver condition | Domain type |
| --- | --- |
| `pgx.ErrNoRows` | `errorx.ErrNotFound` |
| Anything else | `errorx.ErrInternal` |
| `DELETE`/`UPDATE` matching zero rows | `errorx.ErrNotFound` |

That last one matters: a `DELETE` that matches nothing is not a success, even
though the driver reports no error.

---

## Migrations (`migrations/`)

Schema changes are managed with [goose](https://github.com/pressly/goose) and
embedded into the binary with `//go:embed migrations/*.sql`, so a deployed image
carries its own schema history.

Create a new migration:

```sh
goose -dir database/migrations create xxx sql
```

Apply and roll back:

```sh
make migrate-up    # applies every pending migration
make migrate-down  # rolls back the most recent migration only
```

Notes:

- `migrate up` runs with `goose.WithAllowMissing()`. Test seed data shares
  goose's version ledger with the schema migrations, so a schema migration newer
  than a seed would otherwise make the seed look out of order.
- `migrate down` reverts a single migration. To unwind everything, call
  `goose.DownTo(db, dir, 0)` — that is what the round-trip test does.

### Requires PostgreSQL 18+

`20260825120000_profile_id_uuidv7.sql` switches the `profiles` primary key
default from `gen_random_uuid()` to `uuidv7()`, which was added in Postgres 18.

v4 is random, so inserts land at random points in the primary key's B-tree:
every insert dirties a different page and the index fragments as the table
grows. v7 embeds a millisecond timestamp in its high bits, so new rows append to
the right-hand edge of the index instead. Existing rows keep their v4
identifiers; both versions are valid `uuid` values and mix freely in one column.

---

## Testing

Repository tests live in `repositories/tests/` and run against a real Postgres
18 container started by [`pkg/testinfra`](../pkg/testinfra), so behaviour is
verified against the actual database rather than a mock.

`database/migrations_test.go` covers the migrations themselves:

- a full **up → down → up round trip**, which is what proves the up and down
  steps have not drifted apart;
- an assertion that generated ids really are **UUIDv7**;
- a static check that no down step drops a table no up step creates.

Docker must be running. Use `go test -short ./...` to skip the container-backed
suites.
