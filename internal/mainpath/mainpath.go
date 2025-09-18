package mainpath

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

var (
	DataPath   = getDataPath()
	ConfigPath = filepath.Join(xdg.ConfigDirs[0], "cogmoteGO")
)

func getDataPath() string {
	if xdg.DataHome != "" {
		return filepath.Join(xdg.DataHome, "cogmoteGO")
	}

	if len(xdg.DataDirs) > 0 {
		return filepath.Join(xdg.DataDirs[0], "cogmoteGO")
	}

	return filepath.Join(".", "cogmoteGO")
}
