package experiments

import (
	"context"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/mholt/archives"
)

func validateArchiveFormat(file *multipart.FileHeader) bool {
	if filepath.Ext(file.Filename) != ".zip" {
		return false
	} else {
		return true
	}
}

func ArchiveInitExperiment(record ExperimentRecord, file string) error {
	targetDir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)
	if err := os.Mkdir(targetDir, 0755); err != nil {
		return err
	}

	unzip(file, targetDir)
	return nil
}

func ArchiveUpdateExperiment(record ExperimentRecord, file *multipart.FileHeader) {

}

func unzip(file string, targetDir string) error {
	f, _ := os.Open(file)
	var format archives.Zip
	ctx := context.Background()
	err := format.Extract(ctx, f, func(ctx context.Context, info archives.FileInfo) error {
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
