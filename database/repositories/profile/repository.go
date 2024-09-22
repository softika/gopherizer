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
	createSql string
	//go:embed sql/update.sql
	updateSql string
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
	p := new(model.Profile)
	if err := r.db.Pool().QueryRow(ctx, getByIdSql, id).Scan(
		&p.Id,
		&p.AccountId,
		&p.FirstName,
		&p.LastName,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return p, nil
}

func (r Repository) Create(ctx context.Context, p *model.Profile) (*model.Profile, error) {
	if err := r.db.Pool().QueryRow(ctx, createSql,
		p.AccountId, // $1
		p.FirstName, // $2
		p.LastName,  // $3
	).Scan(&p.Id, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}

	return p, nil
}

func (r Repository) Update(ctx context.Context, p *model.Profile) (*model.Profile, error) {
	if err := r.db.Pool().QueryRow(ctx, updateSql,
		p.FirstName, // $1
		p.LastName,  // $2
		p.Id,        // $3
	).Scan(&p.Id, &p.AccountId, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return p, nil
}

func (r Repository) DeleteById(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, deleteByIdSql, id)
	return err
}

func (r Repository) LockById(ctx context.Context, tx pgx.Tx, id string) error {
	var p model.Profile

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

	if err = row.Scan(&p.Id); err != nil {
		return fmt.Errorf("failed to scan profile: %w", err)
	}

	// Ensure no other rows exist
	if row.Next() {
		return fmt.Errorf("multiple users found with id: %v", id)
	}
	return err
}
