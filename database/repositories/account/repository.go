package account

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/softika/gopherizer/database"
	"github.com/softika/gopherizer/internal/account"
	"github.com/softika/gopherizer/internal/role"
	"github.com/softika/gopherizer/pkg/errorx"
)

var (
	//go:embed sql/create.sql
	createSql string
	//go:embed sql/get_by_email.sql
	getByEmailSql string
	//go:embed sql/get_roles.sql
	getRolesSql string
	//go:embed sql/update_password.sql
	updatePasswordSql string
	//go:embed sql/lock_by_id.sql
	lockByIdSql string
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

func (r Repository) Create(ctx context.Context, acc *account.Account) (*account.Account, error) {
	if err := r.Pool().QueryRow(ctx, createSql,
		acc.Email,    // $1
		acc.Password, // $2
	).Scan(&acc.Id, &acc.CreatedAt, &acc.UpdatedAt); err != nil {
		if strings.Contains(err.Error(), "accounts_email_check") {
			return nil, errorx.NewError(
				fmt.Errorf("invalid email %s, error: %w", acc.Email, err),
				errorx.ErrInvalidInput,
			)
		}
		return nil, err
	}

	return acc, nil
}

func (r Repository) GetByEmail(ctx context.Context, email string) (*account.Identity, error) {
	identity := new(account.Identity)

	if err := r.Execute(ctx, func(tx pgx.Tx) error {
		acc := new(account.Account)
		if err := tx.QueryRow(ctx, getByEmailSql, email).Scan(
			&acc.Id,
			&acc.Email,
			&acc.Password,
			&acc.CreatedAt,
			&acc.UpdatedAt,
		); err != nil {
			return errorx.NewError(
				fmt.Errorf("failed to get account by email: %w", err),
				errorx.ErrInvalidInput,
			)
		}

		var roles []string
		rows, err := tx.Query(ctx, getRolesSql, acc.Id)
		if err != nil {
			return fmt.Errorf("failed to get roles: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var rl role.Role
			if err = rows.Scan(&rl.Name); err != nil {
				return fmt.Errorf("failed to scan roles: %w", err)
			}
			roles = append(roles, rl.Name)
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
	return r.Execute(ctx, func(tx pgx.Tx) error {
		acc, err := r.lockById(ctx, tx, id)
		if err != nil {
			return err
		}

		if err = bcrypt.CompareHashAndPassword([]byte(acc.Password), []byte(oldPassword)); err != nil {
			return errorx.NewError(
				fmt.Errorf("invalid old password: %w", err),
				errorx.ErrInvalidInput,
			)
		}

		password, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, updatePasswordSql, password, id)
		if err != nil {
			return errorx.NewError(err, errorx.ErrInvalidInput)
		}

		return nil
	})
}

func (r Repository) lockById(ctx context.Context, tx pgx.Tx, id string) (*account.Account, error) {
	a := new(account.Account)

	row, err := tx.Query(ctx, lockByIdSql, id)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	if !row.Next() {
		return nil, errorx.NewError(
			fmt.Errorf("account with id %s not found", id),
			errorx.ErrNotFound,
		)
	}

	if err = row.Scan(&a.Id, &a.Email, &a.Password); err != nil {
		return nil, fmt.Errorf("failed to scan account: %w", err)
	}

	// ensure no other rows exist
	if row.Next() {
		return nil, fmt.Errorf("multiple accounts found with id: %v", id)
	}

	return a, nil
}
