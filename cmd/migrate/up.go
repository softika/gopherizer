package migrate

import (
	"github.com/spf13/cobra"
	"tldw/database"
	"tldw/logger"

	"tldw/config"
)

var MigrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "runs up database migrations",
	Long:  "runs up database migrations for all storage options defined in go-template",
	Run: func(cmd *cobra.Command, args []string) {
		up()
	},
}

func up() {
	lgr := logger.Get()

	cfg, err := config.New()
	if err != nil {
		lgr.Error("failed to read config", "error", err)
		return
	}

	dvSvc := database.New(cfg.Database)

	lgr.Info("running database migrations")
	if err := migrate(dvSvc.DB()); err != nil {
		lgr.Error("failed to run database migrations", "error", err)
		return
	}

	lgr.Info("database migrations completed successfully")
}
