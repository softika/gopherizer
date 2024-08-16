//go:generate mockgen -source=tx.go -destination=./mock/tx.go -package=mock
package repositories

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tldw/database"
)

type TransactionManager interface {
	Execute(ctx context.Context, fn func(pgx.Tx) error) error
}

type TxManager struct {
	*pgxpool.Pool
}

func NewTxManager(db database.Service) *TxManager {
	return &TxManager{db.Pool()}
}

func (tm *TxManager) Execute(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := tm.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			// re-throw panic after rollbacks
			panic(p)
		} else if err != nil {
			// rollback if error happen
			tx.Rollback(ctx)
		} else {
			// if Commit returns error update err with commit err
			err = tx.Commit(ctx)
		}
	}()
	err = fn(tx)
	return err
}
