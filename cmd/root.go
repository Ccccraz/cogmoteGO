package cmd

import (
	"fmt"
	"os"

	alive "github.com/Ccccraz/cogmoteGO/internal"
	"github.com/Ccccraz/cogmoteGO/internal/broadcast"
	cmdproxy "github.com/Ccccraz/cogmoteGO/internal/cmdProxy"
	"github.com/Ccccraz/cogmoteGO/internal/experiments"
	"github.com/Ccccraz/cogmoteGO/internal/health"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version  = "dev"
	commit   = "test"
	datetime = "unknown"
)

var cfgFile string
var showVersion bool

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

		Serve()
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
	rootCmd.PersistentFlags().BoolVar(&showVersion, "version", false, "show version information")
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

// Default entry point
func Serve() {
	var dev bool

	if envMode := os.Getenv("GIN_MODE"); envMode == "" {
		gin.SetMode(gin.ReleaseMode)
		dev = false
	} else {
		gin.SetMode(envMode)
		dev = true
	}

	logger.Init(dev)
	experiments.Init()

	r := gin.New()

	if gin.Mode() == gin.DebugMode {
		r.Use(gin.Logger())
	} else {
		r.Use(logger.GinMiddleware(logger.Logger))
	}

	r.Use(gin.Recovery())
	r.UseH2C = true

	broadcast.RegisterRoutes(r)
	cmdproxy.RegisterRoutes(r)
	health.RegisterRoutes(r)
	alive.RegisterRoutes(r)
	experiments.RegisterRoutes(r)

	r.Run(":9012")
}

func printVersion() {
	fmt.Printf("cogmoteGO %s (%s %s)", version, commit, datetime)
}
