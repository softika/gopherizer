package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager defines a method to execute a transaction from Begin until Commit or Rollback.
type TxManager interface {
	Begin(context.Context) (pgx.Tx, error)
	Execute(context.Context, func(pgx.Tx) error) error
}

func NewTxManager(db Service) TxManager {
	return &txManager{db.Pool()}
}

type txManager struct {
	*pgxpool.Pool
}

func (tm *txManager) Begin(ctx context.Context) (pgx.Tx, error) {
	return tm.Pool.Begin(ctx)
}

func (tm *txManager) Execute(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := tm.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			err = tx.Rollback(ctx)
			// re-throw panic after rollbacks
			panic(p)
		} else if err != nil {
			// rollback if error happen
			err = tx.Rollback(ctx)
		} else {
			// if Commit returns error update err with commit err
			err = tx.Commit(ctx)
		}
	}()
	err = fn(tx)
	return err
}
