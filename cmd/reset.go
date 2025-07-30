/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/spf13/cobra"
)

// resetCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "quickly reset the service",
	Run: func(cmd *cobra.Command, args []string) {
		service := createService()

		var needreset bool

		if err := service.Uninstall(); err != nil {
			logger.Logger.Info(err.Error())
		} else {
			logger.Logger.Info("Service uninstalled")
			needreset = true
		}

		if err := service.Install(); err != nil {
			logger.Logger.Info(err.Error())
		} else {
			if needreset {
				logger.Logger.Info("Service reinstalled")
			} else {
				logger.Logger.Info("Service installed")
			}
		}

	},
}

func init() {
	serviceCmd.AddCommand(resetCmd)
}
