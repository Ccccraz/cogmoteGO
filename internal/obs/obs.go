package obs

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/Ccccraz/cogmoteGO/internal/commonTypes"
	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/requests/general"
	"github.com/andreykaipov/goobs/api/requests/inputs"
	"github.com/andreykaipov/goobs/api/requests/sceneitems"
	"github.com/andreykaipov/goobs/api/requests/stream"
	"github.com/andreykaipov/goobs/api/typedefs"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/host"
)

// ObsStatus represents basic OBS server info.
type ObsStatus struct {
	ObsVersion string `json:"obs_version"`
	Streaming  bool   `json:"streaming"`
}

// obsData is the input payload for /obs/data
type obsData struct {
	MonkeyName  string  `json:"monkey_name"`
	TrialID     int64   `json:"trial_id"`
	StartTime   string  `json:"start_time"`
	CorrectRate float64 `json:"correct_rate"`
}

var client *goobs.Client

// Default OBS text source name
const obsTextSource = "cogmoteGO"
const obsSence = "cagelab"

func strPtr(s string) *string { return &s }

// getDeviceName automatically retrieves hostname via gopsutil
func getDeviceName() string {
	hostInfo, err := host.Info()
	if err != nil {
		return "unknown"
	}
	return hostInfo.Hostname
}

// Initialize OBS client
func InitObsClient() error {
	var err error
	client, err = goobs.New("localhost:4455") // add goobs.WithPassword("xxx") if needed
	if err != nil {
		return err
	}

	// Check if obsTextSource exists
	inputsList, err := client.Inputs.GetInputList()
	if err != nil {
		return fmt.Errorf("failed to get input list: %w", err)
	}

	exists := false
	for _, input := range inputsList.Inputs {
		if input.InputName == obsTextSource {
			exists = true
			break
		}
	}

	if !exists {
		data := obsData{
			MonkeyName:  "unknown",
			TrialID:     0,
			StartTime:   "unknown",
			CorrectRate: 0,
		}

		formatted := fmt.Sprintf(
			"%s %s %d %.2f%% %s",
			getDeviceName(),
			data.MonkeyName,
			data.TrialID,
			data.CorrectRate*100,
			data.StartTime,
		)

		var inputKind string
		switch runtime.GOOS {
		case "windows":
			inputKind = "text_gdiplus"
		case "linux", "darwin":
			inputKind = "text_ft2_source_v2"
		default:
			return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
		}

		// Create input
		createResp, err := client.Inputs.CreateInput(&inputs.CreateInputParams{
			SceneName: strPtr(obsSence),
			InputName: strPtr(obsTextSource),
			InputKind: strPtr(inputKind),
			InputSettings: map[string]any{
				"text": formatted,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create input %s: %w", obsTextSource, err)
		}

		// Resize and position the source
		_, err = client.SceneItems.SetSceneItemTransform(&sceneitems.SetSceneItemTransformParams{
			SceneName:   strPtr(obsSence),
			SceneItemId: &createResp.SceneItemId,
			SceneItemTransform: &typedefs.SceneItemTransform{
				PositionX:       0.0,
				PositionY:       1080.0,
				ScaleX:          1,
				ScaleY:          1,
				Alignment:       9, // center alignment
				BoundsType:      "OBS_BOUNDS_SCALE_TO_HEIGHT",
				BoundsWidth:     1600.0,
				BoundsHeight:    60.0,
				BoundsAlignment: 9,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to set scene item transform: %w", err)
		}
	}

	return nil
}

// HTTP handler: Initialize OBS connection
func InitObsHandler(c *gin.Context) {
	if err := InitObsClient(); err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to initialize OBS client",
			Detail: err.Error(),
		})
		return
	}
	c.Status(http.StatusCreated)
}

// HTTP handler: Get OBS status
func GetObsStatusHandler(c *gin.Context) {
	if client == nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "OBS client not initialized",
			Detail: "please call /obs/init first",
		})
		return
	}

	version, err := client.General.GetVersion(&general.GetVersionParams{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to get OBS version",
			Detail: err.Error(),
		})
		return
	}

	status, err := client.Stream.GetStreamStatus(&stream.GetStreamStatusParams{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to get OBS stream status",
			Detail: err.Error(),
		})
		return
	}

	resp := ObsStatus{
		ObsVersion: version.ObsVersion,
		Streaming:  status.OutputActive,
	}

	c.JSON(http.StatusOK, resp)
}

// HTTP handler: Start streaming
func PostStartObsStreamingHandler(c *gin.Context) {
	if client == nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "OBS client not initialized",
			Detail: "please call /obs/init first",
		})
		return
	}

	_, err := client.Stream.StartStream(&stream.StartStreamParams{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to start OBS stream",
			Detail: err.Error(),
		})
		return
	}

	c.Status(http.StatusCreated)
}

// HTTP handler: Stop streaming
func PostStopObsStreamingHandler(c *gin.Context) {
	if client == nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "OBS client not initialized",
			Detail: "please call /obs/init first",
		})
		return
	}

	_, err := client.Stream.StopStream(&stream.StopStreamParams{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to stop OBS stream",
			Detail: err.Error(),
		})
		return
	}

	c.Status(http.StatusCreated)
}

// HTTP handler: Update OBS text source with experiment data
func PostObsDataHandler(c *gin.Context) {
	if client == nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "OBS client not initialized",
			Detail: "please call /obs/init first",
		})
		return
	}

	var req obsData
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "invalid request body",
			Detail: err.Error(),
		})
		return
	}

	// Simplified plain data string
	formatted := fmt.Sprintf(
		"%s %s %d %.2f%% %s",
		getDeviceName(),
		req.MonkeyName,
		req.TrialID,
		req.CorrectRate*100,
		req.StartTime,
	)

	_, err := client.Inputs.SetInputSettings(&inputs.SetInputSettingsParams{
		InputName: strPtr(obsTextSource),
		InputSettings: map[string]any{
			"text": formatted,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to set input settings",
			Detail: err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

// Register all OBS routes
func RegisterRoutes(r gin.IRouter) {
	obsGroup := r.Group("/obs")
	{
		obsGroup.GET("", GetObsStatusHandler)
		obsGroup.POST("/init", InitObsHandler)
		obsGroup.POST("/start", PostStartObsStreamingHandler)
		obsGroup.POST("/stop", PostStopObsStreamingHandler)
		obsGroup.POST("/data", PostObsDataHandler)
	}
}
