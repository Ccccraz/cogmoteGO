package cmd

import (
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall cogmoteGO service - need sudo",
	Run: func(cmd *cobra.Command, args []string) {
		service := createService()
		err := service.Uninstall()
		if err != nil {
		    logger.Logger.Info(err.Error())
		} else {
			logger.Logger.Info("Service uninstalled")
		}
	},
}

func init() {
	serviceCmd.AddCommand(uninstallCmd)
}
