package mapper

import (
	"path/filepath"
	"sync"
)

// Map of toolchain targets by module path
type Mapper struct {
	modules map[string][]string
	lock    sync.Mutex
}

func New() *Mapper {
	return &Mapper{
		lock:    sync.Mutex{},
		modules: map[string][]string{},
	}
}

func (t *Mapper) Add(path string, name string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if current, ok := t.modules[path]; ok {
		t.modules[path] = append(current, name)
	} else {
		t.modules[path] = []string{name}
	}
}

// Returns the toolchain module name based on the path.
// The behavior is as follows:
// - For given path, if a toolchain config exists return the first registered toolchain.
// - If no config exists for current path, walk the directories upwards looking for default.
func (t *Mapper) Get(path string) string {
	t.lock.Lock()
	defer t.lock.Unlock()

	for p := path; p != "."; {
		if names, ok := t.modules[p]; ok {
			return names[0]
		}
		p = filepath.Dir(p)
	}

	// Check for root
	if names, ok := t.modules["."]; ok {
		return names[0]
	}

	return ""
}
