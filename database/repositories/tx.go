package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"tldw/database"
)

// Tx defines a transaction interface.
type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// TxManager defines a method to execute a transaction from Begin until Commit or Rollback.
type TxManager interface {
	Begin(context.Context) (Tx, error)
	Execute(context.Context, func(Tx) error) error
}

func NewTxManager(db database.Service) TxManager {
	return &txManager{db.Pool()}
}

type txManager struct {
	*pgxpool.Pool
}

func (tm *txManager) Begin(ctx context.Context) (Tx, error) {
	return tm.Pool.Begin(ctx)
}

func (tm *txManager) Execute(ctx context.Context, fn func(tx Tx) error) error {
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
