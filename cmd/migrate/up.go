package migrate

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"
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
	slog.SetDefault(slogging.Slogger())

	cfg, err := config.New()
	if err != nil {
		slog.Error("failed to read config", "error", err)
		return
	}

	dvSvc := database.New(cfg.Database)

	slog.Info("running database migrations")
	if err := migrate(dvSvc.DB()); err != nil {
		slog.Error("failed to run database migrations", "error", err)
		return
	}

	slog.Info("database migrations completed successfully")
}
