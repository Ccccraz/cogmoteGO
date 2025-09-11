package status

import (
	"net/http"
	"sync"

	"github.com/Ccccraz/cogmoteGO/internal/commonTypes"
	"github.com/Ccccraz/cogmoteGO/internal/experiments"
	"github.com/gin-gonic/gin"
)

var (
		currentStatus = &experiments.ExperimentStatus{
		ID:        "",
		IsRunning: false,
	}
	statusMutex = &sync.Mutex{}
)

func UpdateExperimentStatusHandler(c *gin.Context) {
	var updateData map[string]any
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "failed to bind JSON",
			Detail: err.Error(),
		})
		return
	}

	statusMutex.Lock()
	defer statusMutex.Unlock()

	if id, exist := updateData["id"]; exist {
		if idStr, ok := id.(string); ok {
			currentStatus.ID = idStr
		} else {
			c.JSON(http.StatusBadRequest, commonTypes.APIError{
				Error:  "failed to update id field",
				Detail: "id field must be a string",
			})
			return
		}
	}

	if isRunning, exist := updateData["is_running"]; exist {
		if isRunningBool, ok := isRunning.(bool); ok {
			currentStatus.IsRunning = isRunningBool
		} else {
			c.JSON(http.StatusBadRequest, commonTypes.APIError{
				Error:  "failed to update is_running field",
				Detail: "is_running field must be a boolean",
			})
		}
	}

	c.JSON(http.StatusOK, currentStatus)
}

func GetExperimentStatusHandler(c *gin.Context) {
	statusMutex.Lock()
	defer statusMutex.Unlock()

	c.JSON(http.StatusOK, currentStatus)
}

func RegisterRoutes(r gin.IRouter) {
	r.PATCH("/status", UpdateExperimentStatusHandler)
	r.GET("/status", GetExperimentStatusHandler)
}
