package profile

import (
	"context"
	_ "embed"

	"github.com/softika/gopherizer/database"
	"github.com/softika/gopherizer/internal/profile"
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
)

type Repository struct {
	database.TxManager
	database.Service
}

func NewRepository(db database.Service) Repository {
	return Repository{
		TxManager: database.NewTxManager(db),
		Service:   db,
	}
}

func (r Repository) GetById(ctx context.Context, id string) (*profile.Profile, error) {
	p := new(profile.Profile)
	if err := r.Pool().QueryRow(ctx, getByIdSql, id).Scan(
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

func (r Repository) Create(ctx context.Context, p *profile.Profile) (*profile.Profile, error) {
	if err := r.Pool().QueryRow(ctx, createSql,
		p.AccountId, // $1
		p.FirstName, // $2
		p.LastName,  // $3
	).Scan(&p.Id, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}

	return p, nil
}

func (r Repository) Update(ctx context.Context, p *profile.Profile) (*profile.Profile, error) {
	if err := r.Pool().QueryRow(ctx, updateSql,
		p.FirstName, // $1
		p.LastName,  // $2
		p.Id,        // $3
	).Scan(&p.Id, &p.AccountId, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return p, nil
}

func (r Repository) DeleteById(ctx context.Context, id string) error {
	_, err := r.Pool().Exec(ctx, deleteByIdSql, id)
	return err
}
