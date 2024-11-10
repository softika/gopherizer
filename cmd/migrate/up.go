package migrate

import (
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
	lgr := slogging.Slogger()

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
