package migrate

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"
	"github.com/softika/gopherizer/pkg/logx"
)

var UpCmd = &cobra.Command{
	Use:   "up",
	Short: "runs up database migrations",
	Long:  "runs up database migrations for all storage options defined in go-template",
	Run: func(cmd *cobra.Command, args []string) {
		up()
	},
}

func up() {
	cfg, err := config.New()
	if err != nil {
		slog.Error("failed to read config", "error", err)
		return
	}

	slog.SetDefault(logx.New(cfg.App))

	dvSvc, err := database.New(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return
	}
	defer func() {
		if cErr := dvSvc.Close(); cErr != nil {
			slog.Warn("failed to close database", "error", cErr)
		}
	}()

	slog.Info("running database migrations")
	if err := migrate(dvSvc.DB()); err != nil {
		slog.Error("failed to run database migrations", "error", err)
		return
	}

	slog.Info("database migrations completed successfully")
}
