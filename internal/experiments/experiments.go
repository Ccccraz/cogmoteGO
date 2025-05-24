package experiments

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Ccccraz/cogmoteGO/internal/commonTypes"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/adrg/xdg"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Basic experiment info
type Experiment struct {
	Nickname string `json:"nickname"`
	Type     string `json:"type"`
	Address  string `json:"address"`
	Exec     string `json:"exec"`
}

// Experiment record
type ExperimentRecord struct {
	ID         string `json:"id"`
	Experiment `json:"experiment"`
}

type experimentError struct {
	status  int
	message string
	details string
}

var (
	// all experiment records
	experimentRecords sync.Map

	// current running experiment
	currentExperiment      *os.Process
	currentExperimentMutex sync.Mutex

	// experiments json file
	experimentsJson    string
	experimentsBaseDir string

	logKey = "experiments"
)

func Init() {
	// init experiments json file path
	initPaths()

	// load experiment records from json file if exists
	loadExperimentRecords()
}

// Init experiments json file path
func initPaths() {
	experimentsBaseDir = filepath.Join(xdg.DataHome, "cogmoteGO", "experiments")
	experimentsJson = filepath.Join(experimentsBaseDir, "experiments.json")
	logger.Logger.Debug(
		"location of experiments db file: ",
		slog.Group(
			logKey,
			slog.String("location", experimentsJson),
		),
	)
	logger.Logger.Debug(
		"location of experiments base dir: ",
		slog.Group(
			logKey,
			slog.String("location", experimentsBaseDir),
		),
	)
}

// Save experiment records to json file
func saveExperimentRecord() {
	var data []ExperimentRecord
	experimentRecords.Range(func(key, value any) bool {
		record := value.(ExperimentRecord)
		data = append(data, record)
		return true
	})

	if err := os.MkdirAll(experimentsBaseDir, 0755); err != nil {
		panic(err)
	}

	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(experimentsJson, file, 0644)
	if err != nil {
		panic(err)
	}
}

// Load experiment records from json file
func loadExperimentRecords() {
	if _, err := os.Stat(experimentsJson); os.IsNotExist(err) {
		return
	}

	file, err := os.ReadFile(experimentsJson)
	if err != nil {
		panic(err)
	}

	var data []ExperimentRecord
	err = json.Unmarshal(file, &data)
	if err != nil {
		panic(err)
	}

	for _, record := range data {
		experimentRecords.Store(record.ID, record)
	}
}

// Get Experiments info endpoint
func GetExperiments(c *gin.Context) {
	experiments := make([]ExperimentRecord, 0)

	experimentRecords.Range(func(key, value any) bool {
		experiments = append(experiments, value.(ExperimentRecord))
		return true
	})

	c.JSON(http.StatusOK, experiments)
}

// Get Experiment info by id endpoint
func GetExperimentById(c *gin.Context) {
	id := c.Param("id")

	exp, exists := experimentRecords.Load(id)
	if !exists {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("experiment with id %s not found", id),
			Detail: "",
		})
		return
	}

	c.JSON(http.StatusOK, exp)
}

// Update Experiment info by id endpoint
func UpdateExperimentRecordById(c *gin.Context) {
	id := c.Param("id")

	var experiment Experiment
	if err := c.ShouldBindJSON(&experiment); err != nil {
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "invalid experiment info data",
			Detail: err.Error(),
		})
		return
	}

	record := ExperimentRecord{
		ID:         id,
		Experiment: experiment,
	}

	_, exists := experimentRecords.Load(id)
	experimentRecords.Store(id, record)
	saveExperimentRecord()

	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	c.JSON(status, record)
}

// Delete all experiment records endpoint
func DeleteAllExperimentRecords(c *gin.Context) {
	experimentRecords = sync.Map{}
	saveExperimentRecord()
	c.Status(http.StatusOK)
}

// Delete experiment record by id endpoint
func DeleteExperimentRecordById(c *gin.Context) {
	id := c.Param("id")

	if _, exists := experimentRecords.Load(id); !exists {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("experiment with id %s not found", id),
			Detail: "",
		})
		return
	}

	experimentRecords.Delete(id)
	saveExperimentRecord()
	c.Status(http.StatusOK)
}

// Register new experiment record endpoint
func RegisterExperiment(c *gin.Context) {
	var experiment Experiment
	if err := c.ShouldBindJSON(&experiment); err != nil {
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "invalid experiment info data",
			Detail: err.Error(),
		})
		return
	}

	record := ExperimentRecord{
		ID:         uuid.New().String(),
		Experiment: experiment,
	}

	experimentRecords.Store(record.ID, record)

	// TODO: for binary experiments add init flow
	// Init experiment from github repo by experiment info
	InitExperiment(&record)

	// save experiment record to json file
	saveExperimentRecord()

	c.JSON(http.StatusCreated, record)
}

// Start experiment by id endpoint
func StartExperiment(c *gin.Context) {
	id := c.Param("id")

	// validate experiment record
	record, err := validateExperiment(id)
	if err != nil {
		c.JSON(err.status, commonTypes.APIError{
			Error:  err.message,
			Detail: err.details,
		})
		return
	}

	// check if experiment is already running
	if err := checkExperimentRunning(); err != nil {
		c.JSON(err.status, commonTypes.APIError{
			Error:  err.message,
			Detail: err.details,
		})
		return
	}

	// start experiment process
	process, err := StartExperimentProcess(c.Request.Context(), id, record)
	if err != nil {
		c.JSON(err.status, commonTypes.APIError{
			Error:  err.message,
			Detail: err.details,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "experiment started successfully",
		"pid":     process.Pid,
		"id":      id,
	})
}

// Validate experiment record
func validateExperiment(id string) (*ExperimentRecord, *experimentError) {
	value, exists := experimentRecords.Load(id)
	// check if experiment record exists
	if !exists {
		return nil, &experimentError{
			status:  http.StatusNotFound,
			message: "experiment not found",
			details: fmt.Sprintf("experiment with ID %s not found", id),
		}
	}

	// check if experiment record is valid
	record, ok := value.(ExperimentRecord)
	if !ok {
		return nil, &experimentError{
			status:  http.StatusInternalServerError,
			message: "invalid experiment record type",
			details: fmt.Sprintf("expected ExperimentRecord, got %T", value),
		}
	}

	// check if experiment exec command is empty
	if strings.TrimSpace(record.Exec) == "" {
		return nil, &experimentError{
			status:  http.StatusBadRequest,
			message: "experiment exec command is empty",
			details: fmt.Sprintf("experiment with ID %s has an empty exec command", id),
		}
	}
	return &record, nil
}

// Check if experiment is already running
func checkExperimentRunning() *experimentError {
	currentExperimentMutex.Lock()
	defer currentExperimentMutex.Unlock()

	// check if experiment is already running
	if currentExperiment != nil {
		return &experimentError{
			status:  http.StatusConflict,
			message: "experiment already running",
			details: "another experiment is already running",
		}
	}

	return nil
}

// Start experiment process
func StartExperimentProcess(ctx context.Context, id string, record *ExperimentRecord) (*os.Process, *experimentError) {
	// create working directory for experiment
	workingDir := filepath.Join(experimentsBaseDir, record.Nickname)

	// configure experiment command
	args := strings.Fields(record.Exec)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = workingDir

	// redirect experiment output to stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// start experiment process
	if err := cmd.Start(); err != nil {
		return nil, &experimentError{
			status:  http.StatusInternalServerError,
			message: "failed to start experiment",
			details: err.Error(),
		}
	}

	// check if process is nil
	if cmd.Process == nil {
		return nil, &experimentError{
			status:  http.StatusInternalServerError,
			message: "failed to start experiment",
			details: "process is nil",
		}
	}

	currentExperimentMutex.Lock()
	defer currentExperimentMutex.Unlock()

	// set current experiment as the experiment that was just started
	currentExperiment = cmd.Process

	// start a goroutine to wait for the experiment process to exit
	go func() {
		// wait for the experiment process then initialize the current experiment to nil
		defer func() {
			currentExperimentMutex.Lock()
			currentExperiment = nil
			currentExperimentMutex.Unlock()
		}()

		err := cmd.Wait()
		if err != nil {
			logger.Logger.Error(
				"experiment exited with error: ",
				slog.Group(
					logKey,
					slog.String("id", id),
					slog.String("exit_error", err.Error()),
				),
			)
		}
	}()

	return cmd.Process, nil
}

// Stop experiment by id endpoint
func StopExperiment(c *gin.Context) {
	id := c.Param("id")

	// check if experiment is running
	if currentExperiment == nil {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("no running experiment found with ID %s", id),
			Detail: "",
		})
		return
	}

	// stop experiment process
	if err := currentExperiment.Kill(); err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  fmt.Sprintf("failed to stop experiment with ID %s", id),
			Detail: err.Error(),
		})
		return
	}

	currentExperimentMutex.Lock()
	defer currentExperimentMutex.Unlock()

	// initialize the current experiment to nil
	currentExperiment = nil

	c.JSON(http.StatusOK, gin.H{
		"message": "experiment stopped successfully",
		"id":      id,
	})
}

// Update experiment by git pull endpoint
func UpdateExperiment(c *gin.Context) {
	id := c.Param("id")

	record, ok := experimentRecords.Load(id)
	// check if experiment record exists
	if !ok {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("no experiment found with ID %s", id),
			Detail: "",
		})
		return
	}

	value := record.(ExperimentRecord)

	// initialize experiments working directory
	dir := filepath.Join(experimentsBaseDir, value.Nickname)

	// run git pull command
	cmd := exec.Command("git", "-C", dir, "pull")
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  fmt.Sprintf("failed to update experiment with ID %s", id),
			Detail: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "experiment updated successfully",
		"output":  string(output),
	})
}

// Init experiment by git clone
func InitExperiment(record *ExperimentRecord) {
	// ensure experiments base directory exists
	if err := os.MkdirAll(experimentsBaseDir, 0755); err != nil {
		panic(err)
	}

	// clone experiment repository to experiments base directory with nickname as directory name
	cmd := exec.Command("git", "-C", experimentsBaseDir, "clone", record.Address, record.Nickname)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Logger.Error(
			"experiment failed to initialize: ",
			slog.Group(
				logKey,
				slog.String("nickname", record.Nickname),
				slog.String("init_error", err.Error()),
			),
		)
	}

	logger.Logger.Info(
		"experiment initialized: ",
		slog.Group(
			logKey,
			slog.String("nickname", record.Nickname),
			slog.String("init_log", string(output)),
		),
	)
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/exps", GetExperiments)
	r.GET("/exps/:id", GetExperimentById)
	r.PUT("/exps/:id", UpdateExperimentRecordById)
	r.DELETE("/exps", DeleteAllExperimentRecords)
	r.DELETE("/exps/:id", DeleteExperimentRecordById)
	r.POST("/exps", RegisterExperiment)
	r.POST("/exps/:id/start", StartExperiment)
	r.POST("/exps/:id/stop", StopExperiment)
	r.POST("/exps/:id/update", UpdateExperiment)
}
