package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationsRoundTrip runs up, down and up again.
//
// A down-migration that names tables the up-migration never created is a sign
// the two have drifted apart; the round trip is what proves they still match.
func TestMigrationsRoundTrip(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), dsnFromConfig(dbCfg))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	db := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, goose.SetDialect(GetDialect()))
	goose.SetBaseFS(GetMigrationFS())
	t.Cleanup(func() { goose.SetBaseFS(nil) })

	const dir = "migrations"

	require.NoError(t, goose.Up(db, dir), "up must apply cleanly")
	assert.True(t, tableExists(t, pool, "profiles"), "profiles must exist after up")

	require.NoError(t, goose.Down(db, dir), "down must roll back cleanly")
	assert.False(t, tableExists(t, pool, "profiles"), "profiles must be gone after down")

	require.NoError(t, goose.Up(db, dir), "up must be repeatable after a down")
	assert.True(t, tableExists(t, pool, "profiles"))
}

// TestDownMigrationOnlyDropsWhatUpCreates fails if a down step names a table
// that no up step creates.
func TestDownMigrationOnlyDropsWhatUpCreates(t *testing.T) {
	entries, err := GetMigrationFS().ReadDir("migrations")
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	created := map[string]bool{}
	var dropped []string

	for _, e := range entries {
		body, readErr := GetMigrationFS().ReadFile("migrations/" + e.Name())
		require.NoError(t, readErr)

		up, down := splitGooseSections(string(body))
		for _, tbl := range tablesInCreate(up) {
			created[tbl] = true
		}
		dropped = append(dropped, tablesInDrop(down)...)
	}

	require.NotEmpty(t, created, "expected at least one CREATE TABLE")

	for _, tbl := range dropped {
		assert.Truef(t, created[tbl],
			"down-migration drops %q but no up-migration creates it", tbl)
	}
}

func tableExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()

	var exists bool
	err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`,
		name,
	).Scan(&exists)
	require.NoError(t, err)

	return exists
}
