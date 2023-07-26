package core

import (
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type ModuleMeta struct {
	Srcs           []string `json:"srcs"`
	TransitiveDeps []string `json:"deps"`
}

// Map of metadata keyed by module name.
type BuildMeta map[string]ModuleMeta

var (
	metaData     BuildMeta
	metaDataLock sync.RWMutex
)

func init() {
	metaData = BuildMeta{}
	metaDataLock = sync.RWMutex{}
}

// Collects information about targets.
//
// Currently collects `srcs` and deps.
func metaDataCollector(ctx blueprint.BottomUpMutatorContext) {
	// Alias/defaults are skipped to avoid polluting the file.
	if _, ok := ctx.Module().(*ModuleAlias); ok {
		return
	} else if _, ok := ctx.Module().(*ModuleDefaults); ok {
		return
	}

	meta := ModuleMeta{}

	if s, ok := ctx.Module().(FileConsumer); ok {
		s.GetFiles(ctx).ForEach(
			func(fp file.Path) bool {
				meta.Srcs = append(meta.Srcs, fp.UnScopedPath())
				return true
			})
	}

	ctx.WalkDeps(func(dep, parent blueprint.Module) bool {
		meta.TransitiveDeps = utils.AppendIfUnique(meta.TransitiveDeps, dep.Name())
		return true
	})

	metaDataLock.Lock()
	defer metaDataLock.Unlock()
	metaData[ctx.ModuleName()] = meta
}

// Writes the metadata to specified file if the path is set.
func MetaDataWriteToFile(file string) {
	if file == "" {
		return
	}

	bytes, err := json.Marshal(metaData)
	if err != nil {
		utils.Die("error converting to JSON from: '%v' error: %v", metaData, err)
	}

	err = ioutil.WriteFile(file, bytes, 0644)
	if err != nil {
		utils.Die("error writing to '%s' file: %v", file, err)
	}
}
