package cmd

import (
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop cogmoteGO service",
	Run: func(cmd *cobra.Command, args []string) {
		service := createService()
		err := service.Stop()
		if err != nil {
		    logger.Logger.Info(err.Error())
		} else {
			logger.Logger.Info("Service stopped")
		}
	},
}

func init() {
	serviceCmd.AddCommand(stopCmd)
}
