package cmd

import (
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start cogmoteGO service",
	Run: func(cmd *cobra.Command, args []string) {
		service, _ := createService()
		err := service.Start()
		if err != nil {
			logger.Logger.Info(err.Error())
		} else {
			logger.Logger.Info("Service started")
		}
	},
}

func init() {
	serviceCmd.AddCommand(startCmd)
}
