package experiments

// experimentRecord
type ExperimentRecord struct {
	// Registration ID of experiment in cogmoteGO
	ID string `json:"id"`
	// The status of the experiments repository
	Status string `json:"status"`
	// The current branch of the Git type repository
	Branch *string `json:"branch"`
	// The time of registration to cogmoteGO
	RegisterTime string `json:"register_time"`
	// Last update time
	LastUpdate string `json:"last_update"`
	// Experiment meta-information
	Experiment Experiment `json:"experiment"`
}

// Experiment meta-information
//
// experiment
type Experiment struct {
	// The name of the experiment registered to cogmoteGO
	Nickname string `json:"nickname"`
	// The existence form of experimental files
	Type string `json:"type"`
	// If it is a git repository, then the address of the repository is
	Address *string `json:"address"`
	// Experimental data path
	DataPath *string `json:"data_path"`
	// Commands that cogmoteGO is expected to execute when accessing the start port
	Execs []Exec `json:"execs"`
}

type Exec struct {
	// The name of the command
	Nickname *string `json:"nickname"`
	// Specific commands
	Exec string `json:"exec"`
}

type Status string

const (
	Uninitialized Status = "uninitialized"
	Ok            Status = "ok"
)

type ExperimentType string

const (
	Git     ExperimentType = "git"
	Archive ExperimentType = "archive"
	Local   ExperimentType = "local"
)
