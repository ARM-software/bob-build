package plugin

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	bob "github.com/ARM-software/bob-build/core"
	"github.com/google/blueprint"
)

type bobParser struct {
	rootPath string
	config   *bob.BobConfig
}

func newBobParser(rootPath string, config *bob.BobConfig) *bobParser {
	return &bobParser{
		rootPath: rootPath,
		config:   config,
	}
}

func (p *bobParser) parse() {
	// We only need the Blueprint context to parse the modules, we discard it afterwards.
	bp := blueprint.NewContext()

	// register all Bob's module types
	bob.RegisterModuleTypes(func(name string, mf bob.FactoryWithConfig) {
		// Create a closure passing the config to a module factory so
		// that the module factories can access the config.
		factory := func() (blueprint.Module, []interface{}) {
			return mf(p.config)
		}
		bp.RegisterModuleType(name, factory)
	})

	// resolve defaults with bob built-ins
	bp.RegisterBottomUpMutator("default_deps1", bob.DefaultDepsStage1Mutator).Parallel()
	bp.RegisterBottomUpMutator("default_deps2", bob.DefaultDepsStage2Mutator).Parallel()
	bp.RegisterBottomUpMutator("default_applier", bob.DefaultApplierMutator).Parallel()

	bpToParse, err := findBpFiles(p.rootPath)
	if err != nil {
		log.Fatalf("Creating bplist failed: %v\n", err)
	}

	_, errs := bp.ParseFileList(p.rootPath, bpToParse, nil)

	if len(errs) > 0 {
		for _, e := range errs {
			log.Printf("Parse failed: %v\n", e)
		}
		os.Exit(1)
	}

	_, errs = bp.ResolveDependencies(nil)

	if len(errs) > 0 {
		for _, e := range errs {
			log.Printf("Dependency failed: %v\n", e)
		}
		os.Exit(1)
	}
}

func findBpFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if filepath.Base(path) == "build.bp" {
			files = append(files, path)
			return nil
		}

		return nil
	})
	return files, err
}
