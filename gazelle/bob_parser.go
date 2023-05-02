package plugin

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"text/scanner"

	bob "github.com/ARM-software/bob-build/core"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

const (
	ConditionDefault string = "//conditions:default"
)

type valueHandler func(feature string, attribute string, v interface{})

type bobParser struct {
	rootPath     string
	BobIgnoreDir []string
	config       *bob.BobConfig
}

func newBobParser(rootPath string, BobIgnoreDir []string, config *bob.BobConfig) *bobParser {
	return &bobParser{
		rootPath:     rootPath,
		BobIgnoreDir: BobIgnoreDir,
		config:       config,
	}
}

func (p *bobParser) parse() []*Module {
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

	var modules []*Module
	var modulesMap map[string]*Module = make(map[string]*Module)
	var modulesMutex sync.RWMutex

	bp.RegisterBottomUpMutator("register_bob_modules", func(mctx blueprint.BottomUpMutatorContext) {
		m := NewModule(mctx.ModuleName(), mctx.ModuleType(), mctx.ModuleDir(), p.rootPath)

		parseBpModule(mctx.Module(), func(feature string, attribute string, v interface{}) {
			m.addFeatureAttribute(feature, attribute, v)
		})

		modulesMutex.Lock()
		defer modulesMutex.Unlock()
		modulesMap[mctx.ModuleName()] = m
		modules = append(modules, m)
	}).Parallel()

	bpToParse, err := findBpFiles(p.rootPath, p.BobIgnoreDir)
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

	// set proper indexes for all the parsed modules
	bp.VisitAllModulesWithPos(func(m blueprint.Module, p scanner.Position) {
		if mod, ok := modulesMap[m.Name()]; ok {
			mod.SetIndex(uint32(p.Line))
		}
	})

	return modules
}

func findBpFiles(root string, ignoreDirs []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		for _, ignoreDir := range ignoreDirs {
			if strings.TrimPrefix(path, root+"/") == ignoreDir {
				return filepath.SkipDir
			}
		}

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

func parseBpModule(m blueprint.Module, handler valueHandler) {
	// get module's `Properties`
	if f, ok := m.(bob.PropertyProvider); ok {
		parseBpModuleProperties(f.GetProperties(), handler)
	} else {
		log.Printf("module '%s' does not implement 'bob.PropertyProvider'\n", reflect.TypeOf(m))
	}
}

func parseBpModuleProperties(v interface{}, handler valueHandler) {
	propType := reflect.TypeOf(v)
	if propType.Kind() == reflect.Ptr {
		propType = propType.Elem()
	}

	propValue := reflect.ValueOf(v)
	if propValue.Kind() == reflect.Ptr {
		propValue = propValue.Elem()
	}

	for i := 0; i < propType.NumField(); i++ {
		if propType.Field(i).IsExported() {
			name := propType.Field(i).Name
			fieldValue := propValue.FieldByName(name).Interface()
			if propValue.Field(i).Kind() == reflect.Struct && name == "Features" {
				parseBpModuleFeatures(fieldValue, handler)
			} else {
				parseProperties(ConditionDefault, name, fieldValue, handler)
			}
		}
	}
}

func parseBpModuleFeatures(v interface{}, handler valueHandler) {

	embedFeaturesPtr := reflect.ValueOf(v).FieldByName("BlueprintEmbed").Interface()
	embedFeaturesValue := reflect.ValueOf(embedFeaturesPtr).Elem()
	embedFeaturesType := reflect.TypeOf(embedFeaturesPtr).Elem()

	// iterate every feature inside passed `f`
	for i := 0; i < embedFeaturesType.NumField(); i++ {
		featureName := embedFeaturesType.Field(i).Name
		feature := embedFeaturesValue.FieldByName(featureName).FieldByName("BlueprintEmbed").Interface()
		featureType := reflect.TypeOf(feature)

		if featureType.Kind() == reflect.Pointer {
			featureType = featureType.Elem()
		}

		// `feature` has to be of `reflect.Struct` kind
		if featureType.Kind() == reflect.Struct {
			featureValue := reflect.ValueOf(feature)
			for j := 0; j < featureType.NumField(); j++ {
				propertyName := featureType.Field(j).Name
				propertyValue := reflect.Indirect(featureValue).FieldByName(propertyName).Interface()

				parseProperties(featureName, propertyName, propertyValue, handler)
			}
		}
	}
}

func parseProperties(featureName string, propertyName string, v interface{}, handler valueHandler) {

	propType := reflect.TypeOf(v)

	if propType == reflect.TypeOf(bob.TargetSpecific{}) {
		// TODO Property struct `bob.TargetSpecific` not supported
		return
	}

	if propType.Kind() == reflect.Struct {
		structValue := reflect.ValueOf(v)

		// iterate all struct fields
		for i := 0; i < propType.NumField(); i++ {
			if propType.Field(i).IsExported() {
				structField := propType.Field(i)
				structFieldName := structField.Name

				// ignore `blueprint:"mutated"` fields
				if proptools.HasTag(structField, "blueprint", "mutated") {
					continue
				}

				fieldValue := structValue.FieldByName(structFieldName).Interface()

				parseProperties(featureName, structFieldName, fieldValue, handler)
			}
		}
	} else {
		propFound := checkSimpleType(propertyName, v)
		if propFound {
			handler(featureName, propertyName, v)
		}
	}
}

func checkSimpleType(propertyName string, v interface{}) bool {
	var ret bool = false

	switch v.(type) {
	case []string:
		if len(v.([]string)) != 0 {
			ret = true
		}
	case *string:
		if v.(*string) != nil && *v.(*string) != "" {
			ret = true
		}
	case *bool:
		if v.(*bool) != nil {
			ret = true
		}
	case string:
		if v.(string) != "" {
			ret = true
		}
	case bob.TgtType:
		if v.(bob.TgtType) != "" {
			ret = true
		}

	// Internal types which are only used in Bob
	// TODO: ideally we should check for the blueprint:mutated tag on struct properties and ignore those.
	case bob.FilePaths:
		// ignore
	default:
		log.Printf("Unhandled type:  %s \n  attribute: %s \n", reflect.TypeOf(v), propertyName)
	}

	return ret
}
