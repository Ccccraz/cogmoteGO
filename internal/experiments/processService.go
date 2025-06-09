package experiments

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Ccccraz/cogmoteGO/internal/logger"
)

type ProcessService struct {
	// current running experiment
	currentExperiment      *os.Process
	currentExperimentMutex sync.Mutex
}

func validateExecs(record ExperimentRecord) error {
	// check if experiment exec command is empty
	if len(record.Experiment.Execs) == 0 {
		return fmt.Errorf("experiment exec command is empty")
	}
	return nil
}

// Check if experiment is already running
func (ps *ProcessService) checkExperimentRunning() error {
	ps.currentExperimentMutex.Lock()
	defer ps.currentExperimentMutex.Unlock()

	// check if experiment is already running
	if ps.currentExperiment != nil {
		return fmt.Errorf("experiment already running")
	}

	return nil
}

// Start experiment process
func (ps *ProcessService) StartExperimentProcess(ctx context.Context, id string, record ExperimentRecord) (*os.Process, error) {
	// create working directory for experiment
	workingDir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)

	// configure experiment command
	args := strings.Fields(record.Experiment.Execs[0].Exec)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = workingDir

	// redirect experiment output to stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// start experiment process
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// check if process is nil
	if cmd.Process == nil {
		return nil, fmt.Errorf("process is nil")
	}

	ps.currentExperimentMutex.Lock()
	defer ps.currentExperimentMutex.Unlock()

	// set current experiment as the experiment that was just started
	ps.currentExperiment = cmd.Process

	// start a goroutine to wait for the experiment process to exit
	go func() {
		// wait for the experiment process then initialize the current experiment to nil
		defer func() {
			ps.currentExperimentMutex.Lock()
			ps.currentExperiment = nil
			ps.currentExperimentMutex.Unlock()
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
