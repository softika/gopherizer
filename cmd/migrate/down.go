package migrate

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"
	"github.com/softika/gopherizer/pkg/logx"
)

var DownCmd = &cobra.Command{
	Use:   "down",
	Short: "rollback database migrations",
	Long:  "rollback database migrations for all tables",
	Run: func(cmd *cobra.Command, args []string) {
		down()
	},
}

func down() {
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

	slog.Info("rollback database migrations")
	if err := rollback(dvSvc.DB()); err != nil {
		slog.Error("failed to rollback database migrations", "error", err)
		return
	}

	slog.Info("database migrations rollback completed successfully")
}
