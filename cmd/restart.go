package cmd

import (
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "restarts cogmoteGO service",
	Run: func(cmd *cobra.Command, args []string) {
		service, _ := createService()
		err := service.Restart()
		if err != nil {
			logger.Logger.Info(err.Error())
		} else {
			logger.Logger.Info("Service restarted successfully")
		}
	},
}

func init() {
	serviceCmd.AddCommand(restartCmd)
}
