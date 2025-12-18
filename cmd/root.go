package cmd

import (
	"fmt"
	"os"

	"github.com/Ccccraz/cogmoteGO/internal/config"
	"github.com/spf13/cobra"
)

var (
	version  = "dev"
	commit   = "test"
	datetime = "unknown"

	cfgFile     string
	Config      config.Config
	showVersion bool
	showVerbose bool
)

var rootCmd = &cobra.Command{
	Use:   "cogmoteGO",
	Short: "Air Traffic Control for Remote Neuroexperiments",
	Long: `
	cogmoteGO is the 'air traffic control' for remote neuroexperiments: 
	a lightweight Go system coordinating distributed data streams, commands,
	and full experiment lifecycle management - from deployment to data collection.
	`,

	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			printVersion()
			return
		}

		service, _ := createService()
		service.Run()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.config/cogmoteGO/config.toml)")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "show version information")
	rootCmd.PersistentFlags().BoolVar(&showVerbose, "verbose", false, "verbose output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	Config = config.LoadConfig(cfgFile)
}

func printVersion() {
	fmt.Printf("cogmoteGO %s (%s %s)\n", version, commit, datetime)
}
