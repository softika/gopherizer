package migrate

import (
	"github.com/spf13/cobra"
	"tldw/config"
	"tldw/database"
	"tldw/logging"
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
	lgr := logging.Logger()

	cfg, err := config.New()
	if err != nil {
		lgr.Error("failed to read config", "error", err)
		return
	}

	dvSvc := database.New(cfg.Database)

	lgr.Info("rollback database migrations")
	if err := rollback(dvSvc.DB()); err != nil {
		lgr.Error("failed to rollback database migrations", "error", err)
		return
	}

	lgr.Info("database migrations rollback completed successfully")
}
