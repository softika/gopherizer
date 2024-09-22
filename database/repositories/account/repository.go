package account

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5"

	"tldw/database"
	"tldw/database/repositories"
	"tldw/internal/model"
)

var (
	//go:embed sql/create.sql
	createSql string
	//go:embed sql/get_by_email.sql
	getByEmailSql string
	//go:embed sql/get_roles.sql
	getRolesSql string
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

func (r Repository) Create(ctx context.Context, acc *model.Account) (*model.Account, error) {
	if err := r.db.Pool().QueryRow(ctx, createSql,
		acc.Email,    // $1
		acc.Password, // $2
	).Scan(&acc.Id, &acc.CreatedAt, &acc.UpdatedAt); err != nil {
		return nil, err
	}

	return acc, nil
}

func (r Repository) GetByEmail(ctx context.Context, email string) (*model.Identity, error) {
	identity := new(model.Identity)

	if err := r.Execute(ctx, func(tx pgx.Tx) error {
		acc := new(model.Account)
		if err := tx.QueryRow(ctx, getByEmailSql, email).Scan(
			&acc.Id,
			&acc.Email,
			&acc.Password,
			&acc.CreatedAt,
			&acc.UpdatedAt,
		); err != nil {
			return fmt.Errorf("failed to get account by email: %w", err)
		}

		var roles []string
		rows, err := tx.Query(ctx, getRolesSql, acc.Id)
		if err != nil {
			return fmt.Errorf("failed to get roles: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var role model.Role
			if err = rows.Scan(&role.Name); err != nil {
				return fmt.Errorf("failed to scan roles: %w", err)
			}
			roles = append(roles, role.Name)
		}

		identity.AccountId = acc.Id
		identity.Email = acc.Email
		identity.Password = acc.Password
		identity.Roles = roles

		return nil
	}); err != nil {
		return nil, err
	}

	return identity, nil
}

func (r Repository) ChangePassword(ctx context.Context, id string, oldPassword string, newPassword string) error {
	return nil
}
