package repositories

import (
	"context"
	_ "embed"

	"github.com/softika/gopherizer/database"
	"github.com/softika/gopherizer/internal/profile"
)

var (
	//go:embed sql/profile_get_by_id.sql
	profileGetByIdSql string
	//go:embed sql/profile_insert.sql
	profileInsertSql string
	//go:embed sql/profile_update.sql
	profileUpdateSql string
	//go:embed sql/profile_delete_by_id.sql
	profileDeleteByIdSql string
)

type ProfileRepository struct {
	database.TxManager
	database.Service
}

func NewProfileRepository(db database.Service) ProfileRepository {
	return ProfileRepository{
		TxManager: database.NewTxManager(db),
		Service:   db,
	}
}

func (r ProfileRepository) GetById(ctx context.Context, id string) (*profile.Profile, error) {
	p := new(profile.Profile)
	if err := r.Pool().QueryRow(ctx, profileGetByIdSql, id).Scan(
		&p.Id,
		&p.FirstName,
		&p.LastName,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return p, nil
}

func (r ProfileRepository) Create(ctx context.Context, p *profile.Profile) (*profile.Profile, error) {
	if err := r.Pool().QueryRow(ctx, profileInsertSql,
		p.FirstName, // $1
		p.LastName,  // $2
	).Scan(&p.Id, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}

	return p, nil
}

func (r ProfileRepository) Update(ctx context.Context, p *profile.Profile) (*profile.Profile, error) {
	if err := r.Pool().QueryRow(ctx, profileUpdateSql,
		p.FirstName, // $1
		p.LastName,  // $2
		p.Id,        // $3
	).Scan(&p.Id, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return p, nil
}

func (r ProfileRepository) DeleteById(ctx context.Context, id string) error {
	_, err := r.Pool().Exec(ctx, profileDeleteByIdSql, id)
	return err
}
