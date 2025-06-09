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
	// ensure experiments base directory exists
	if err := os.MkdirAll(experimentsBaseDir, 0755); err != nil {
		return nil, err
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
	// initialize experiments working directory
	dir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)

	// run git pull command
	cmd := exec.Command("git", "-C", dir, "pull")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return output, nil
}

func gitSwitch(record ExperimentRecord, branch string) ([]byte, error) {
	// initialize experiments working directory
	dir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)

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
