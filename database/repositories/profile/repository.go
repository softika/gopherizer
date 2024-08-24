package profile

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/jackc/pgx/v5"
	"tldw/database"
	"tldw/database/repositories"
	"tldw/internal/errorx"
	"tldw/internal/model"
)

var (
	//go:embed sql/get_by_id.sql
	getByIdSql string
	//go:embed sql/create.sql
	createUserSql string
	//go:embed sql/update.sql
	updateUserSql string
	//go:embed sql/delete_by_id.sql
	deleteByIdSql string
	//go:embed sql/lock_by_id.sql
	lockByIdSql string
)

type Repository struct {
	repositories.TxManager
	db database.Service
}

func NewRepository(dbService database.Service) Repository {
	return Repository{
		TxManager: repositories.NewTxManager(dbService),
		db:        dbService,
	}
}

func (r Repository) GetById(ctx context.Context, id string) (*model.Profile, error) {
	u := new(model.Profile)
	if err := r.db.Pool().
		QueryRow(ctx, getByIdSql, id).
		Scan(
			&u.Id,
			&u.FirstName,
			&u.LastName,
			&u.Email,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
		return nil, err
	}

	return u, nil
}

func (r Repository) Create(ctx context.Context, u *model.Profile) (*model.Profile, error) {
	err := r.db.Pool().QueryRow(ctx, createUserSql,
		u.FirstName, // $1
		u.LastName,  // $2
		u.Email,     // $3
	).Scan(&u.Id, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (r Repository) Update(ctx context.Context, u *model.Profile) (*model.Profile, error) {
	err := r.db.Pool().QueryRow(ctx, updateUserSql,
		u.FirstName, // $1
		u.LastName,  // $2
		u.Email,     // $3
		u.Id,        // $4
	).Scan(&u.Id, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r Repository) DeleteById(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, deleteByIdSql, id)
	return err
}

func (r Repository) LockById(ctx context.Context, tx pgx.Tx, id string) error {
	var u model.Profile

	row, err := tx.Query(ctx, lockByIdSql, id)
	if err != nil {
		return err
	}
	defer row.Close()

	if !row.Next() {
		return errorx.NewError(
			fmt.Errorf("profile with id %s not found", id),
			errorx.ErrNotFound,
		)
	}

	if err = row.Scan(&u.Id); err != nil {
		return fmt.Errorf("failed to scan profile: %w", err)
	}

	// Ensure no other rows exist
	if row.Next() {
		return fmt.Errorf("multiple users found with id: %v", id)
	}
	return err
}
