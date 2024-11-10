package cmd

import (
	"github.com/spf13/cobra"

	"github.com/softika/gopherizer/cmd/serve"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve command",
	Long:  `serve command runs the server`,
	Run: func(cmd *cobra.Command, args []string) {
		serve.Run()
	},
}
