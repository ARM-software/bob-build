package parser

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"text/scanner"

	bob "github.com/ARM-software/bob-build/core"
	bob_file "github.com/ARM-software/bob-build/core/file"
	bob_toolchain "github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/gazelle/common"
	"github.com/ARM-software/bob-build/gazelle/mapper"
	mod "github.com/ARM-software/bob-build/gazelle/module"
	"github.com/ARM-software/bob-build/gazelle/util"
	"github.com/google/blueprint"
	"github.com/google/blueprint/parser"
	"github.com/google/blueprint/proptools"
)

type valueHandler func(feature string, attribute string, v interface{})

type Parser struct {
	rootPath     string         // TODO: remove
	relPath      string         // TODO: remove
	BobIgnoreDir []string       // TODO: remove
	config       *bob.BobConfig // TODO: remove

	m     *mapper.Mapper
	scope *parser.Scope
}

func (p *Parser) GetScope() *parser.Scope {
	return p.scope
}

func NewLegacy(rootPath string, relPath string, BobIgnoreDir []string, config *bob.BobConfig) *Parser {
	return &Parser{
		rootPath:     rootPath,
		relPath:      relPath,
		BobIgnoreDir: BobIgnoreDir,
		config:       config,
	}
}

func New(
	m *mapper.Mapper,
	parent *parser.Scope) *Parser {

	return &Parser{
		m:     m,
		scope: parser.NewScope(parent),
	}
}

func (p *Parser) Parse(rootPath, pkgPath, fileName string) (*parser.File, error) {
	absolute := filepath.Join(rootPath, pkgPath, fileName)

	f, err := os.OpenFile(absolute, os.O_RDONLY, 0400)
	if err != nil {
		log.Fatalf("Failed to read file:'%v' with error:'%v'\n", absolute, err)
		return nil, err
	}

	ast, errs := parser.ParseAndEval(filepath.Join(pkgPath, "build.bp"), f, p.scope)
	if len(errs) != 0 {
		return nil, fmt.Errorf("Failed to parse blueprint file %v with %v", absolute, errs)
	}

	// Given a valid AST, register all module targets in mapper
	for _, def := range ast.Defs {
		switch def := def.(type) {
		case *parser.Module:
			if prop, ok := def.GetProperty("name"); ok {
				target := prop.Value.(*parser.String).Value
				label := mapper.MakeLabel(target, pkgPath)
				p.m.Map(label, target)
			}
		}
	}

	return ast, nil
}

func (p *Parser) ParseLegacy() []*mod.Module {
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

	var modules []*mod.Module
	var modulesMap map[string]*mod.Module = make(map[string]*mod.Module)
	var modulesMutex sync.RWMutex

	bp.RegisterBottomUpMutator("register_bob_modules", func(ctx blueprint.BottomUpMutatorContext) {
		m := mod.NewModule(
			ctx.ModuleName(),
			ctx.ModuleType(),
			filepath.Join(p.relPath, ctx.ModuleDir()),
			p.rootPath)

		parseBpModule(ctx.Module(), func(feature string, attribute string, v interface{}) {
			m.AddFeatureAttribute(feature, attribute, v)
		})

		modulesMutex.Lock()
		defer modulesMutex.Unlock()
		modulesMap[ctx.ModuleName()] = m
		modules = append(modules, m)
	}).Parallel()

	bpToParse, err := FindBpFiles(p.rootPath, p.relPath, p.BobIgnoreDir)
	if err != nil {
		log.Fatalf("Creating bplist failed: %v\n", err)
	}

	bob.SetupLogger(nil)
	_, errs := bp.ParseFileList(filepath.Join(p.rootPath, p.relPath), bpToParse, nil)

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

func FindBpFiles(root string, rel string, ignoreDirs []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(filepath.Join(root, rel), func(path string, d fs.DirEntry, err error) error {

		rel, _ := filepath.Rel(root, path)

		for _, ignoreDir := range ignoreDirs {
			if ignore, _ := util.IsChildFilepath(ignoreDir, rel); ignore {
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
				parseProperties(common.ConditionDefault, name, fieldValue, handler)
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
	case bool:
		if v.(bool) == true || v.(bool) == false {
			ret = true
		}
	case string:
		if v.(string) != "" {
			ret = true
		}
	case bob_toolchain.TgtType:
		if v.(bob_toolchain.TgtType) != "" {
			ret = true
		}

	// Internal types which are only used in Bob
	// TODO: ideally we should check for the blueprint:mutated tag on struct properties and ignore those.
	case bob_file.Paths:
		// ignore
	default:
		log.Printf("Unhandled type:  %s \n  attribute: %s \n", reflect.TypeOf(v), propertyName)
	}

	return ret
}
