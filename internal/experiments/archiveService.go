package experiments

import (
	"archive/zip"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ccccraz/cogmoteGO/internal/commonTypes"
	"github.com/gin-gonic/gin"
)

func validateArchiveFormat(file *multipart.FileHeader) bool {
	if filepath.Ext(file.Filename) != ".zip" {
		return false
	} else {
		return true
	}
}

func ArchiveInitExperiment(record ExperimentRecord, source string) error {
	dstDir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)
	if err := os.Mkdir(dstDir, 0755); err != nil {
		return err
	}

	if err := unzip(source, dstDir); err != nil {
		return err
	}

	return nil
}

func ArchiveUpdateExperiment(record ExperimentRecord, source string) error {
	dstDir := filepath.Join(experimentsBaseDir, record.Experiment.Nickname)

	if _, err := os.Stat(dstDir); err == nil {
		if err := os.RemoveAll(dstDir); err != nil {
			return fmt.Errorf("failed to remove existing directory: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check directory status: %v", err)
	}

	if err := ArchiveInitExperiment(record, source); err != nil {
		return err
	}

	return nil
}

func unzip(source string, destination string) error {
	// open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	// iterate over the files in the archive and extract them
	for _, file := range reader.File {
		err := unzipFile(file, destination)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(file *zip.File, destination string) error {
	// check if file paths are not vulnerable to Zip Slip attack
	filePath := filepath.Join(destination, file.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	if file.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}

		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	zippedFile, err := file.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}

	return nil
}

func validateArchiveMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		record := repo.load(id)
		if record.Experiment.Type != "archive" {
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
