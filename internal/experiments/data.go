package experiments

import (
	"net/http"
	"os"
	"sync"
)

type DataFs struct {
	dirs []string
	mu   sync.RWMutex
}

func (d *DataFs) Open(name string) (http.File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, dir := range d.dirs {
		fs := http.Dir(dir)
		if f, err := fs.Open(name); err == nil {
			return f, nil
		}
	}
	return nil, os.ErrNotExist
}

func (d *DataFs) Add(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.dirs = append(d.dirs, name)
}
