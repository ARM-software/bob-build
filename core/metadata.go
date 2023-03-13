package core

import (
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type ModuleMeta struct {
	TransitiveSrcs []string `json:"srcs"`
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

// Helper to extract sources from various types of modules.
func (m *ModuleMeta) extractSrcs(module blueprint.Module) {
	if s, ok := module.(sourceInterface); ok {
		m.TransitiveSrcs = utils.AppendUnique(m.TransitiveSrcs, s.getSourceFiles(nil))
	} else if s, ok := module.(getGenerateCommonInterface); ok {
		gc := s.getGenerateCommon()
		m.TransitiveSrcs = utils.AppendUnique(m.TransitiveSrcs, gc.Properties.Tools)
		m.TransitiveSrcs = utils.AppendUnique(m.TransitiveSrcs, gc.Properties.Srcs)
	}
}

// Collects information about targets.
//
// Currently collects `srcs` and deps.
func metaDataCollector(mctx blueprint.BottomUpMutatorContext) {
	// Alias/defaults are skipped to avoid polluting the file.
	if _, ok := mctx.Module().(*alias); ok {
		return
	} else if _, ok := mctx.Module().(*defaults); ok {
		return
	}

	meta := ModuleMeta{}
	meta.extractSrcs(mctx.Module())

	mctx.WalkDeps(func(dep, parent blueprint.Module) bool {
		meta.TransitiveDeps = utils.AppendIfUnique(meta.TransitiveDeps, dep.Name())
		meta.extractSrcs(dep)
		return true
	})

	metaDataLock.Lock()
	defer metaDataLock.Unlock()
	metaData[mctx.ModuleName()] = meta
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
