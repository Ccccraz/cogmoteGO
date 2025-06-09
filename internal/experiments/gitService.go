package experiments

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Ccccraz/cogmoteGO/internal/commonTypes"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/gin-gonic/gin"
)

func gitInitExperiment(record ExperimentRecord) ([]byte, error) {
	// check if experiment is uninitialized
	// if not, return error
	if record.Status != string(Uninitialized) {
		return nil, fmt.Errorf("experiment is already initialized")
	}

	// ensure experiments base directory exists
	if err := os.MkdirAll(experimentsBaseDir, 0755); err != nil {
		return nil, err
	}

	// check if experiment directory already exists
	// if it does, remove it
	dstDir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)
	if _, err := os.Stat(dstDir); err == nil {
		if err := os.RemoveAll(dstDir); err != nil {
			return nil, fmt.Errorf("failed to remove existing directory: %v", err)
		}
	}

	// clone experiment repository to experiments base directory with nickname as directory name
	cmd := exec.Command("git", "-C", experimentsBaseDir, "clone", *record.Experiment.Address, record.Experiment.Nickname)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Logger.Error(
			"experiment failed to initialize: ",
			slog.Group(
				logKey,
				slog.String("nickname", record.Experiment.Nickname),
				slog.String("init_error", err.Error()),
			),
		)

		return nil, err
	}

	logger.Logger.Info(
		"experiment initialized: ",
		slog.Group(
			logKey,
			slog.String("nickname", record.Experiment.Nickname),
			slog.String("init_log", string(output)),
		),
	)

	return output, nil
}

func gitUpdateExperiment(record ExperimentRecord) ([]byte, error) {
	// check if experiment is initialized
	if record.Status == string(Uninitialized) {
		return nil, fmt.Errorf("experiment is uninitialized")
	}

	// check if experiment directory exists
	dir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("experiment directory does not exist")
	}

	// run git pull command
	cmd := exec.Command("git", "-C", dir, "pull")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return output, nil
}

func gitSwitch(record ExperimentRecord, branch string) ([]byte, error) {
	// check if experiment is initialized
	if record.Status == string(Uninitialized) {
		return nil, fmt.Errorf("experiment is uninitialized")
	}

	// initialize experiments working directory
	dir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("experiment directory does not exist")
	}

	// run git switch command
	cmd := exec.Command("git", "-C", dir, "switch", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return output, nil
}

func validateGitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		record := repo.load(id)
		if record.Experiment.Type != "git" {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				commonTypes.APIError{
					Error:  "experiment type is not git",
					Detail: fmt.Sprintf("experiment type is %s", record.Experiment.Type),
				},
			)

			return
		}

		c.Next()
	}
}
