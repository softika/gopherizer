package migrate

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"github.com/softika/gopherizer/database"
)

// migrate runs all migration scripts inside the migrations folder.
func migrate(db *sql.DB) error {
	if err := goose.SetDialect(database.GetDialect()); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	goose.SetBaseFS(database.GetMigrationFS())
	if err := goose.Up(db, "migrations", goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	return nil
}

// rollback runs all migration scripts inside the migrations folder.
func rollback(db *sql.DB) error {
	if err := goose.SetDialect(database.GetDialect()); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	goose.SetBaseFS(database.GetMigrationFS())
	if err := goose.Down(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	return nil
}
