package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newTestTxManager builds a TxManager on a dedicated pool.
// It deliberately avoids New(), whose sync.Once singleton is shared with (and
// closed by) the other tests in this package.
func newTestTxManager(t *testing.T) (TxManager, *pgxpool.Pool) {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), dsnFromConfig(dbCfg))
	if err != nil {
		t.Fatalf("failed to create test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return NewTxManager(&service{pool: pool}), pool
}

func TestTxManagerExecuteCommits(t *testing.T) {
	tm, pool := newTestTxManager(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `CREATE TABLE tx_commit_test (id INT PRIMARY KEY)`); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS tx_commit_test`)
	})

	err := tm.Execute(ctx, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, `INSERT INTO tx_commit_test (id) VALUES (1)`)
		return execErr
	})
	if err != nil {
		t.Fatalf("expected Execute to succeed, got %v", err)
	}

	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM tx_commit_test`).Scan(&count); err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 committed row, got %d", count)
	}
}

func TestTxManagerExecuteRollsBackOnError(t *testing.T) {
	tm, pool := newTestTxManager(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `CREATE TABLE tx_rollback_test (id INT PRIMARY KEY)`); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS tx_rollback_test`)
	})

	wantErr := errors.New("business logic failed")

	err := tm.Execute(ctx, func(tx pgx.Tx) error {
		if _, execErr := tx.Exec(ctx, `INSERT INTO tx_rollback_test (id) VALUES (1)`); execErr != nil {
			return execErr
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected Execute to return %v, got %v", wantErr, err)
	}

	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM tx_rollback_test`).Scan(&count); err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after rollback, got %d", count)
	}
}

// TestTxManagerExecuteReturnsCommitError guards the case where the callback
// succeeds but COMMIT itself fails. A deferred unique constraint defers the
// violation to commit time, which is exactly when the error must surface.
func TestTxManagerExecuteReturnsCommitError(t *testing.T) {
	tm, pool := newTestTxManager(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		CREATE TABLE tx_commit_error_test (
			id INT PRIMARY KEY,
			val INT,
			CONSTRAINT uniq_val UNIQUE (val) DEFERRABLE INITIALLY DEFERRED
		)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS tx_commit_error_test`)
	})

	err = tm.Execute(ctx, func(tx pgx.Tx) error {
		// Both inserts succeed; the unique violation is deferred until COMMIT.
		if _, execErr := tx.Exec(ctx, `INSERT INTO tx_commit_error_test (id, val) VALUES (1, 42)`); execErr != nil {
			return execErr
		}
		_, execErr := tx.Exec(ctx, `INSERT INTO tx_commit_error_test (id, val) VALUES (2, 42)`)
		return execErr
	})

	if err == nil {
		t.Fatal("expected Execute to return the COMMIT error, got nil " +
			"(commit failed but the caller was told the transaction succeeded)")
	}

	// The data must not be visible, since the commit was rejected.
	var count int
	if qErr := pool.QueryRow(ctx, `SELECT count(*) FROM tx_commit_error_test`).Scan(&count); qErr != nil {
		t.Fatalf("failed to count rows: %v", qErr)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after failed commit, got %d", count)
	}
}
