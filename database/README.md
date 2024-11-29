# Database Package Documentation

The database package provides essential functionality for interacting with a Postgres database, including connection management, transaction handling, and repositories. 
It also includes a migrations folder containing SQL scripts to manage schema changes.

---

## Components

### Database Service
- **Purpose**:  
  - Manages the connection pool to the Postgres database using [pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v4/pgxpool).
- **Features**:
    - Configurable parameters (host, port, user, password, database, connection pool size, etc.).
    - Connection health checks.


### Migrations
- **Purpose**:  
  - Simplifies database transaction management by abstracting the process of starting, committing, and rolling back transactions.
- **Key Features:**:
  - Implements a TxManager interface with:
    - `Begin(ctx context.Context)`: Starts a new transaction.
    - `Execute(ctx context.Context, func(pgx.Tx) error)`: Executes a transaction block with automatic commit/rollback logic.
- **Error and Panic Handling**:
    - Ensures rollback in case of errors or panics during transaction execution.
    - Re-throws panics after rollback to preserve original error flow.
- **Example Usage**:
```go
type ProfileRepository struct {
    database.TxManager
    database.Service
}

func (r ProfileRepository) UpdateWithTx(ctx context.Context, p *profile.Profile) (*profile.Profile, error) {
    // add more db operations here like lock by id or other operations

    err := r.Execute(ctx, func(tx pgx.Tx) error {
        if err := tx.QueryRow(ctx, profileUpdateSql,
            p.FirstName, // $1
            p.LastName,  // $2
            p.Id,        // $3
        ).Scan(&p.Id, &p.CreatedAt, &p.UpdatedAt); err != nil {
            return err
        }
        return nil
    })
return p, err
}
```

### Repositories
- **Purpose**:  
  Encapsulates data access logic to keep it separate from business logic.
- **Features**:
    - Implements repository patterns for entities, e.g., `UserRepository`, `ProfileRepository`.
    - Each repository interacts with the database using the connection pool or a transaction context.
