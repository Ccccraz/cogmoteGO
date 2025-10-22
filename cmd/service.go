package cmd

import (
	"os"
	"os/user"
	"runtime"

	alive "github.com/Ccccraz/cogmoteGO/internal"
	"github.com/Ccccraz/cogmoteGO/internal/broadcast"
	cmdproxy "github.com/Ccccraz/cogmoteGO/internal/cmdProxy"
	"github.com/Ccccraz/cogmoteGO/internal/device"
	"github.com/Ccccraz/cogmoteGO/internal/email"
	"github.com/Ccccraz/cogmoteGO/internal/experiments"
	"github.com/Ccccraz/cogmoteGO/internal/health"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/Ccccraz/cogmoteGO/internal/obs"
	"github.com/Ccccraz/cogmoteGO/internal/status"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var (
	password string
	usermode = false
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
		service, config := createService()

		if config.Option["UserService"] != nil || config.UserName != "" {
			logger.Logger.Info("Installing as user service")
		} else {
			logger.Logger.Info("Installing as system service")
		}

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
	serviceCmd.Flags().BoolVarP(&usermode, "user", "u", false, "install service as user service")

	if runtime.GOOS == "windows" {
		serviceCmd.Flags().StringVarP(&password, "password", "p", "", "when installing as user service, provide the password")
	}
}

func createService() (service.Service, service.Config) {
	logger.Init(true)
	options := make(service.KeyValue)

	svcConfig := &service.Config{
		Name:        "cogmoteGO",
		DisplayName: "cogmoteGO",
		Description: "cogmoteGO is the 'air traffic control' for remote neuroexperiments: a lightweight Go system coordinating distributed data streams, commands, and full experiment lifecycle management - from deployment to data collection.",
		Option:      options,
	}

	switch runtime.GOOS {
	case "windows":
		if usermode {
			username, err := user.Current()
			if err != nil {
				logger.Logger.Info(err.Error())
			}
			svcConfig.UserName = username.Username
			svcConfig.Option["Password"] = password
		}
		svcConfig.Option["OnFailure"] = "restart"
		svcConfig.Option["OnFailureDelayDuration"] = "60s"

	case "linux":
		if usermode {
			svcConfig.Option["UserService"] = true
		} else {
			svcConfig.Dependencies = []string{
				"After=network.target",
			}
		}
	case "darwin":
		if usermode {
			svcConfig.Option["UserService"] = true
		} else {
		}
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Logger.Info(err.Error())
	}
	return s, *svcConfig
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
		return origin == "http://localhost:1420" || origin == "http://localhost:5173" || origin == "tauri://localhost" || origin == "http://tauri.localhost"
	}
	r.Use(cors.New(corsConfig))

	api := r.Group("/api")

	broadcast.RegisterRoutes(api)
	cmdproxy.RegisterRoutes(api)
	health.RegisterRoutes(api)
	alive.RegisterRoutes(api)
	experiments.RegisterRoutes(api)
	status.RegisterRoutes(api)
	device.SetVersion(version, commit, datetime)
	device.RegisterRoutes(api)
	obs.RegisterRoutes(api)
	email.RegisterRoutes(api)

	r.Run(":9012")
}
