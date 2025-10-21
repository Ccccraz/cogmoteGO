package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version  = "dev"
	commit   = "test"
	datetime = "unknown"

	cfgFile     string
	showVersion bool
	showVerbose bool
)

type Config struct {
	SendEmail string `mapstructure:"send_email"`
}

var appConfig Config

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
	viper.SetConfigType("toml")
	viper.SetDefault("send_email", "")
	viper.AutomaticEnv()

	var configPath string

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		configPath = cfgFile
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to resolve home directory: %v\n", err)
			return
		}
		configPath = filepath.Join(home, ".config", "cogmoteGO", "config.toml")
		viper.SetConfigFile(configPath)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create config directory: %v\n", err)
		return
	}

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		viper.Set("send_email", viper.GetString("send_email"))
		if writeErr := viper.WriteConfigAs(configPath); writeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to create config: %v\n", writeErr)
			return
		}
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "failed to check config file: %v\n", err)
		return
	} else if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config: %v\n", err)
		return
	}

	if err := viper.Unmarshal(&appConfig); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse config: %v\n", err)
	}
}

func printVersion() {
	fmt.Printf("cogmoteGO %s (%s %s)\n", version, commit, datetime)
}
