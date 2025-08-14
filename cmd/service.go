package cmd

import (
	"os"
	"os/user"
	"runtime"

	alive "github.com/Ccccraz/cogmoteGO/internal"
	"github.com/Ccccraz/cogmoteGO/internal/broadcast"
	cmdproxy "github.com/Ccccraz/cogmoteGO/internal/cmdProxy"
	"github.com/Ccccraz/cogmoteGO/internal/device"
	"github.com/Ccccraz/cogmoteGO/internal/experiments"
	"github.com/Ccccraz/cogmoteGO/internal/health"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var (
	password string
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
		serviceCmd.Flags().StringVarP(&password, "password", "p", "", "install service with password")
		serviceCmd.MarkFlagRequired("password")
	}
}

func createService() service.Service {
	logger.Init(true)
	options := make(service.KeyValue)

	svcConfig := &service.Config{
		Name:        "cogmoteGO",
		DisplayName: "cogmoteGO",
		Description: "cogmoteGO is the 'air traffic control' for remote neuroexperiments: a lightweight Go system coordinating distributed data streams, commands, and full experiment lifecycle management - from deployment to data collection.",
		Option:      options,
	}

	if runtime.GOOS == "windows" {
		username, err := user.Current()
		if err != nil {
			logger.Logger.Info(err.Error())
		}
		svcConfig.UserName = username.Username

		svcConfig.Option["Password"] = password
		svcConfig.Option["OnFailure"] = "restart"
		svcConfig.Option["OnFailureDelayDuration"] = "60s"

		logger.Logger.Info("Service will be installed as user: " + username.Username)
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
	dev := showVerbose

	envMode := os.Getenv("GIN_MODE")
	if envMode == "" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(envMode)
		dev = dev || envMode == gin.DebugMode
	}

	logger.Init(dev)
	experiments.Init()

	r := gin.New()
	if dev {
		r.Use(gin.Logger())
	} else {
		r.Use(logger.GinMiddleware(logger.Logger))
	}

	r.Use(gin.Recovery())
	r.UseH2C = true

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOriginFunc = func(origin string) bool {
		return origin == "http://localhost:1420" || origin == "tauri://localhost"
	}
	r.Use(cors.New(corsConfig))

	api := r.Group("/api")

	broadcast.RegisterRoutes(api)
	cmdproxy.RegisterRoutes(api)
	health.RegisterRoutes(api)
	alive.RegisterRoutes(api)
	experiments.RegisterRoutes(api)
	device.RegisterRoutes(api)

	r.Run(":9012")
}
