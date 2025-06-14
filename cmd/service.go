package cmd

import (
	"os"
	"runtime"

	alive "github.com/Ccccraz/cogmoteGO/internal"
	"github.com/Ccccraz/cogmoteGO/internal/broadcast"
	cmdproxy "github.com/Ccccraz/cogmoteGO/internal/cmdProxy"
	"github.com/Ccccraz/cogmoteGO/internal/experiments"
	"github.com/Ccccraz/cogmoteGO/internal/health"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var (
	user      string
	password  string
)

type program struct {
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	Serve()
}

func (p *program) Stop(s service.Service) error {
	return nil
}

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "install cogmoteGO as a service",
	Run: func(cmd *cobra.Command, args []string) {
		service := createService()
		err := service.Install()
		if err != nil {
			logger.Logger.Info(err.Error())
		} else {
			logger.Logger.Info("Service installed")
		}
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	if runtime.GOOS == "windows" {
		serviceCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "install service for user")
		serviceCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "install service with password")
	}
}

func createService() service.Service {
	logger.Init(true)
	options := make(service.KeyValue)

	svcConfig := &service.Config{
		Name:        "cogmoteGO",
		DisplayName: "cogmoteGO",
		Description: "cogmoteGO is the 'air traffic control' for remote neuroexperiments: a lightweight Go system coordinating distributed data streams, commands, and full experiment lifecycle management - from deployment to data collection.",
		UserName:    user,
		Option:      options,
	}

	if runtime.GOOS == "windows" {
		svcConfig.Option["Password"] = password
		svcConfig.Option["OnFailure"] = "restart"
		svcConfig.Option["OnFailureDelayDuration"] = "60s"
	}

	if runtime.GOOS == "linux" {
		svcConfig.Dependencies = []string{
			"After=network.target",
		}
		svcConfig.Option["UserService"] = true
	}

	if runtime.GOOS == "darwin" {
		svcConfig.Option["UserService"] = true
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Logger.Info(err.Error())
	}

	return s
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
