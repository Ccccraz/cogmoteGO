package experiments

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/adrg/xdg"
)

var (
	// experiments json file
	experimentsJson    string
	experimentsBaseDir string
)

type Repository struct {
	// all experiment records
	experimentRecords sync.Map
}

func Init() {
	// init experiments json file path
	repo.initPaths()

	// load experiment records from json file if exists
	repo.loadJson()
}

// Init experiments json file path
func (r *Repository) initPaths() {
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

func (r *Repository) Store(record ExperimentRecord) {
	r.experimentRecords.Store(record.ID, record)
	r.saveJson()
}

func (r *Repository) Delete(id string) {
	r.experimentRecords.Delete(id)
	r.saveJson()
}

func (r *Repository) Range(f func(key, value any) bool) {
	r.experimentRecords.Range(f)
}

func (r *Repository) LoadAll() []ExperimentRecord {
	experiments := make([]ExperimentRecord, 0)

	r.experimentRecords.Range(func(key, value any) bool {
		experiments = append(experiments, value.(ExperimentRecord))
		return true
	})

	return experiments
}

func (r *Repository) Clear() {
	r.experimentRecords.Clear()
}

// load and validate experiment record
func (r *Repository) load(id string) ExperimentRecord {
	value, _ := r.experimentRecords.Load(id)

	// check if experiment record is valid
	record := value.(ExperimentRecord)

	return record
}

func (r *Repository) LoadAndDelete(id string) ExperimentRecord {
	value, _ := r.experimentRecords.LoadAndDelete(id)
	r.saveJson()

	// check if experiment record is valid
	record, _ := value.(ExperimentRecord)

	return record
}

// Save experiment records to json file
func (r *Repository) saveJson() {
	var data []ExperimentRecord
	r.experimentRecords.Range(func(key, value any) bool {
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
func (r *Repository) loadJson() {
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
		r.experimentRecords.Store(record.ID, record)
	}
}

func (r *Repository) validateIfExperimentExists(id string) bool {
	_, exists := r.experimentRecords.Load(id)
	return exists
}
