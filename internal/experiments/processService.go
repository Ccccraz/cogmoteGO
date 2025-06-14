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

func (ps *ProcessService) StartLocalExperimentProcess(ctx context.Context, id string, record ExperimentRecord, nickname *string) (*os.Process, error) {
	if record.Experiment.Address == nil || *record.Experiment.Address == "" {
		return nil, fmt.Errorf("experiment address is empty")
	}

	workingDir, err := filepath.Abs(*record.Experiment.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist", workingDir)
	}

	return ps.StartProcess(ctx, id, record, workingDir, nickname)
}

func (ps *ProcessService) StartExperimentProcess(ctx context.Context, id string, record ExperimentRecord, nickname *string) (*os.Process, error) {
	if record.Experiment.Nickname == "" {
		return nil, fmt.Errorf("experiment nickname is empty")
	}

	// create working directory for experiment
	workingDir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)

	if err := os.MkdirAll(experimentsBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %v", err)
	}

	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist", workingDir)
	}

	return ps.StartProcess(ctx, id, record, workingDir, nickname)
}

// Start experiment process
func (ps *ProcessService) StartProcess(ctx context.Context, id string, record ExperimentRecord, workingDir string, nickname *string) (*os.Process, error) {
	// configure experiment command
	var args []string
	var cmd *exec.Cmd
	if nickname != nil {
		for _, e := range record.Experiment.Execs {
			if e.Nickname == nickname {
				args = strings.Fields(e.Exec)
				cmd = exec.Command(args[0], args[1:]...)
			}
		}
	} else {
		args = strings.Fields(record.Experiment.Execs[0].Exec)
		cmd = exec.Command(args[0], args[1:]...)
	}

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
