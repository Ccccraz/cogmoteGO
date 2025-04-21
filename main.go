package main

import (
	"os"

	"github.com/Ccccraz/cogmoteGO/internal/broadcast"
	cmdproxy "github.com/Ccccraz/cogmoteGO/internal/cmdProxy"
	"github.com/Ccccraz/cogmoteGO/internal/health"
	"github.com/gin-gonic/gin"
)

func main() {
	if envMode := os.Getenv("GIN_MODE"); envMode == "" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(envMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.UseH2C = true

	if gin.Mode() == gin.DebugMode {
		r.Use(gin.Logger())
	}

	broadcast.RegisterRoutes(r)
	cmdproxy.RegisterRoutes(r)
	health.RegisterRoutes(r)

	r.Run(":9012")
}
