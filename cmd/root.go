package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version  = "dev"
	commit   = "test"
	datetime = "unknown"

	cfgFile string
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

		service := createService()
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cogmoteGO.toml)")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "show version information")
	rootCmd.PersistentFlags().BoolVar(&showVerbose, "verbose", false, "verbose output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cogmoteGO" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("toml")
		viper.SetConfigName(".cogmoteGO")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func printVersion() {
	fmt.Printf("cogmoteGO %s (%s %s)\n", version, commit, datetime)
}
