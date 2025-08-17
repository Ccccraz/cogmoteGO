package device

import (
	"net/http"
	"os/user"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
)

var (
	version  string
	commit   string
	datetime string
)

type Device struct {
	Username string `json:"username"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	CPUModel string `json:"cpu"`
	Uptime   uint64 `json:"uptime"`
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	Datetime string `json:"datetime"`
}

func SetVersion(v string, c string, d string) {
	version = v
	commit = c
	datetime = d
}

func GetHealth(c *gin.Context) {
	hostInfo, _ := host.Info()
	cpuInfo, _ := cpu.Info()
	user, _ := user.Current()

	var cpuModel string
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	healthReport := &Device{
		Username: user.Username,
		Hostname: hostInfo.Hostname,
		OS:       hostInfo.OS,
		Arch:     hostInfo.KernelArch,
		CPUModel: cpuModel,
		Uptime:   hostInfo.Uptime,
		Version:  version,
		Commit:   commit,
		Datetime: datetime,
	}

	c.JSON(http.StatusOK, healthReport)
}

func RegisterRoutes(r gin.IRouter) {
	r.GET("/device", GetHealth)
}
