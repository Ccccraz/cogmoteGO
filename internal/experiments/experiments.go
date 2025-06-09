package experiments

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Ccccraz/cogmoteGO/internal/commonTypes"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	repo           *Repository
	processService *ProcessService
	logKey         = "experiments"
)

// Get Experiments info endpoint
func GetExperimentRecordsHandler(c *gin.Context) {
	experiments := repo.LoadAll()

	c.JSON(http.StatusOK, experiments)
}

// Register new experiment record endpoint
func RegisterExperimentHandler(c *gin.Context) {
	var experiment Experiment

	if err := c.ShouldBindJSON(&experiment); err != nil {
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "invalid experiment info data",
			Detail: err.Error(),
		})
		return
	}

	now := time.Now()
	record := ExperimentRecord{
		ID:           uuid.New().String(),
		Status:       string(Uninitialized),
		RegisterTime: now.String(),
		LastUpdate:   now.String(),
		Experiment:   experiment,
	}

	repo.Store(record)

	c.JSON(http.StatusCreated, record)
}

// Delete all experiment records endpoint
func DeleteAllExperimentRecordsHandler(c *gin.Context) {
	repo.Clear()

	c.Status(http.StatusOK)
}

// Update Experiment info by id endpoint
func UpdateExperimentRecordHandler(c *gin.Context) {
	id := c.Param("id")

	var experiment Experiment
	if err := c.ShouldBindJSON(&experiment); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "invalid experiment info",
			Detail: err.Error(),
		})
		return
	}

	record := repo.LoadAndDelete(id)

	record.LastUpdate = time.Now().String()
	record.Experiment = experiment

	repo.Store(record)
	c.JSON(http.StatusOK, record)
}

// Get Experiment info by id endpoint
func GetExperimentRecordHandler(c *gin.Context) {
	id := c.Param("id")

	record := repo.load(id)

	c.JSON(http.StatusOK, record)
}

// Delete experiment record by id endpoint
func DeleteExperimentRecordHandler(c *gin.Context) {
	id := c.Param("id")

	repo.Delete(id)
	c.Status(http.StatusOK)
}

// Start experiment by id endpoint
func StartExperimentHandler(c *gin.Context) {
	id := c.Param("id")

	// validate experiment record
	record := repo.load(id)
	if err := validateExecs(record); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to validate experiment exec command",
			Detail: err.Error(),
		})
		return
	}

	// check if experiment is already running
	if err := processService.checkExperimentRunning(); err != nil {
		c.AbortWithStatusJSON(http.StatusConflict, commonTypes.APIError{
			Error:  "experiment already running",
			Detail: err.Error(),
		})
		return
	}

	// start experiment process
	process, err := processService.StartExperimentProcess(c.Request.Context(), id, record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to start experiment process",
			Detail: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "experiment started successfully",
		"pid":     process.Pid,
		"id":      id,
	})
}

// Stop experiment by id endpoint
func StopExperimentHandler(c *gin.Context) {
	id := c.Param("id")

	// check if experiment is running
	if processService.currentExperiment == nil {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("no running experiment found with ID %s", id),
			Detail: "",
		})
		return
	}

	// stop experiment process
	if err := processService.currentExperiment.Kill(); err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  fmt.Sprintf("failed to stop experiment with ID %s", id),
			Detail: err.Error(),
		})
		return
	}

	processService.currentExperimentMutex.Lock()
	defer processService.currentExperimentMutex.Unlock()

	// initialize the current experiment to nil
	processService.currentExperiment = nil

	c.JSON(http.StatusOK, gin.H{
		"message": "experiment stopped successfully",
		"id":      id,
	})
}

// Update experiment by git pull endpoint
func GitUpdateExperimentHandler(c *gin.Context) {
	id := c.Param("id")
	record := repo.LoadAndDelete(id)
	defer repo.Store(record)

	output, err := gitUpdateExperiment(record)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  fmt.Sprintf("failed to update experiment with ID %s", id),
			Detail: err.Error(),
		})
		return
	}

	record.LastUpdate = time.Now().String()

	c.JSON(http.StatusOK, gin.H{
		"message": "experiment updated successfully",
		"output":  string(output),
	})
}

// Init experiment by git clone
func GitInitExperimentHandler(c *gin.Context) {
	id := c.Param("id")
	record := repo.LoadAndDelete(id)
	defer repo.Store(record)

	output, err := gitInitExperiment(record)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to initialize experiment",
			Detail: err.Error(),
		})
		return
	}

	record.Status = string(Ok)

	c.JSON(http.StatusOK, gin.H{
		"message": "experiment initialized successfully",
		"output":  string(output),
	})
}

func GitExperimentBranchHandler(c *gin.Context) {
	id := c.Param("id")
	branch := c.Param("branch")

	record := repo.LoadAndDelete(id)
	defer repo.Store(record)

	output, err := gitSwitch(record, branch)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  fmt.Sprintf("failed to switch experiment branch to %s", branch),
			Detail: err.Error(),
		})
		return
	}

	record.Branch = &branch

	c.JSON(http.StatusOK, gin.H{
		"message": "experiment branch switched successfully",
		"output":  string(output),
	})
}

func ArchiveExperimentInitHandler(c *gin.Context) {
	id := c.Param("id")
	record := repo.LoadAndDelete(id)
	defer repo.Store(record)

	file, err := c.FormFile("file")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "failed to get file from request",
			Detail: err.Error(),
		})
		return
	}

	if !validateArchiveFormat(file) {
		c.AbortWithStatusJSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "invalid archive format",
			Detail: "currently only zip is supported",
		})
	}

	tempDir, err := os.MkdirTemp(experimentsBaseDir, "upload-")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to create temporary directory",
			Detail: err.Error(),
		})
	}
	defer os.RemoveAll(tempDir)

	if err := c.SaveUploadedFile(file, tempDir); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to save uploaded file",
			Detail: err.Error(),
		})
	}

	ArchiveInitExperiment(record, tempDir)

	record.Status = string(Ok)
}

func ArchiveExperimentUpdateHandler(c *gin.Context) {
}

func validateIfExperimentExistsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if !repo.validateIfExperimentExists(id) {
			c.AbortWithStatusJSON(http.StatusNotFound, commonTypes.APIError{
				Error:  fmt.Sprintf("experiment with ID %s not found", id),
				Detail: "",
			})
			return
		}

		c.Next()
	}
}

func RegisterRoutes(r *gin.Engine) {
	expGroup := r.Group("/exps")
	{
		expGroup.GET("", GetExperimentRecordsHandler)
		expGroup.POST("", RegisterExperimentHandler)
		expGroup.DELETE("", DeleteAllExperimentRecordsHandler)

		idGroup := expGroup.Group("/:id")
		idGroup.Use(validateIfExperimentExistsMiddleware())
		{
			idGroup.GET("", GetExperimentRecordHandler)
			idGroup.PUT("", UpdateExperimentRecordHandler)
			idGroup.DELETE("", DeleteExperimentRecordHandler)

			r.POST("/git", GitInitExperimentHandler)
			r.PUT("/git", GitUpdateExperimentHandler)
			r.POST("/git/:branch", GitExperimentBranchHandler)

			r.POST("/artifacts", ArchiveExperimentInitHandler)
			r.PUT("/artifacts", ArchiveExperimentUpdateHandler)

			r.POST("/start", StartExperimentHandler)
			r.POST("/stop", StopExperimentHandler)
		}
	}
}
