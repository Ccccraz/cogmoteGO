package health

import (
	"net/http"
	"os/user"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/host"
)

type HealthReport struct {
	Status   string `json:"status"`
	Username string `json:"username"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Uptime   uint64 `json:"uptime"`
}

func GetHealth(c *gin.Context) {
	info, _ := host.Info()
	user, _ := user.Current()

	healthReport := &HealthReport{
		Status:   "running",
		Username: user.Username,
		Hostname: info.Hostname,
		OS:       info.OS,
		Arch:     info.KernelArch,
		Uptime:   info.Uptime,
	}

	c.JSON(http.StatusOK, healthReport)
}

func RegisterRoutes(r gin.IRouter) {
	r.GET("/health", GetHealth)
}
