package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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

// Execute runs fn inside a transaction, committing when fn returns nil and
// rolling back otherwise.
//
// The result is named on purpose: the deferred func decides between commit and
// rollback, and an unnamed result would be copied out before that runs, hiding
// a failed COMMIT from the caller.
func (tm *txManager) Execute(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	tx, err := tm.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tm.rollback(ctx, tx)
			// re-throw panic after rollback
			panic(p)
		}

		if err != nil {
			// Keep the original failure; a rollback error is additional context,
			// never a replacement.
			if rbErr := tm.rollbackErr(ctx, tx); rbErr != nil {
				err = errors.Join(err, rbErr)
			}
			return
		}

		if cErr := tx.Commit(ctx); cErr != nil {
			err = fmt.Errorf("failed to commit transaction: %w", cErr)
		}
	}()

	err = fn(tx)
	return err
}

// rollbackErr rolls back tx, ignoring the benign "already closed" case.
func (tm *txManager) rollbackErr(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("failed to roll back transaction: %w", err)
	}
	return nil
}

// rollback rolls back tx and logs any failure, for paths that cannot return it.
func (tm *txManager) rollback(ctx context.Context, tx pgx.Tx) {
	if err := tm.rollbackErr(ctx, tx); err != nil {
		slog.ErrorContext(ctx, "rollback failed", "error", err)
	}
}
