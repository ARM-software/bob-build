package core

import (
	"reflect"
	"strings"
	"text/template"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
)

// Currently only supported for Legacy source props
type matchSourceInterface interface {
	getLegacySourceProperties() *LegacySourceProps
	getMatchSourcePropNames() []string
}

// Insert a function callback for a specific property.
func addtoFuncmap(propfnmap map[string]template.FuncMap, propList []string, name string,
	fn interface{}) {

	for _, prop := range propList {
		if _, ok := propfnmap[prop]; !ok {
			propfnmap[prop] = make(template.FuncMap)
		}
		propfnmap[prop][name] = fn
	}
}

// Apply late templates on strings, slices and recursively in structures.
//
// This function supports property specific funcmaps for templates,
// allowing template functions to only be valid for particular
// properties.
func applyLateTemplateRecursive(propsVal reflect.Value, stringvalues map[string]string,
	propfnmap map[string]template.FuncMap) {

	for i := 0; i < propsVal.NumField(); i++ {
		field := propsVal.Field(i)
		propName := propsVal.Type().Field(i).Name

		switch field.Kind() {
		case reflect.String:
			if funcmap, ok := propfnmap[propName]; ok {
				applyTemplateString(field, stringvalues, funcmap)
			}

		case reflect.Slice:
			// Array of strings
			if funcmap, ok := propfnmap[propName]; ok {
				emptyStrings := false
				for j := 0; j < field.Len(); j++ {
					elem := field.Index(j)
					if elem.Kind() == reflect.String {
						applyTemplateString(elem, stringvalues, funcmap)
						if elem.String() == "" {
							emptyStrings = true
						}
					}
				}

				if emptyStrings {
					// The template expansion has left empty
					// strings in the slice, so strip them
					list := field.Interface().([]string)
					list = stripEmptyComponents(list)
					field.Set(reflect.ValueOf(list))
				}
			}

		case reflect.Ptr:
			if funcmap, ok := propfnmap[propName]; ok {
				tgtField := reflect.Indirect(field)
				if tgtField.Kind() == reflect.String {
					applyTemplateString(tgtField, stringvalues, funcmap)
				}
			}

		case reflect.Struct:
			applyLateTemplateRecursive(field, stringvalues, propfnmap)
		}
	}
}

// Record non-compiled sources (only relevant for C/C++ compiled
// libraries/binaries)
func (s *LegacySourceProps) initializeNonCompiledSourceMap(ctx blueprint.BaseModuleContext) map[string]bool {
	// Unused non-compiled sources are not allowed, so create
	// a map to mark whether a non-compiled source is matched.
	nonCompiledSources := make(map[string]bool)
	if _, ok := getLibrary(ctx.Module()); ok {
		s.GetFiles(ctx).ForEachIf(
			func(fp file.Path) bool {
				return !fp.IsType(file.TypeCompilable)
			},
			func(fp file.Path) bool {
				nonCompiledSources[fp.ScopedPath()] = false

				return true
			})
	}
	return nonCompiledSources
}

// Set up {{match_srcs}} handling
//
// {{match_srcs}} returns the result of the input glob when applied to
// the modules source list. Because it needs access to the source
// list, this runs much later than other templates.
//
// This template is only applied in specific properties where we've
// seen sensible use-cases:
// - Build Props:
//   - Ldflags
//
// - Generated Common:
//   - Args
//   - Cmd
func setupMatchSources(ctx blueprint.BaseModuleContext,
	propfnmap map[string]template.FuncMap) map[string]bool {

	var sourceProps *LegacySourceProps
	var matchSrcProps []string

	if m, ok := ctx.Module().(matchSourceInterface); ok {
		sourceProps = m.getLegacySourceProperties()
		matchSrcProps = m.getMatchSourcePropNames()
	}

	nonCompiledSources := sourceProps.initializeNonCompiledSourceMap(ctx)
	addtoFuncmap(propfnmap, matchSrcProps, "match_srcs",
		func(arg string) string {
			return sourceProps.matchSources(ctx, arg, nonCompiledSources)
		})

	return nonCompiledSources
}

// Callback function implementing {{match_srcs}}
func (s *LegacySourceProps) matchSources(ctx blueprint.BaseModuleContext, arg string,
	matchedNonCompiledSources map[string]bool) string {

	matchedSources := []string{}

	s.GetFiles(ctx).ForEach(
		func(fp file.Path) bool {
			matched, err := pathtools.Match("**/"+arg, fp.ScopedPath())
			if err != nil {
				utils.Die("Error during matching filepath pattern")
			}
			if matched {
				matchedNonCompiledSources[fp.ScopedPath()] = true
				matchedSources = append(matchedSources, fp.BuildPath())
			}

			return true
		})

	if len(matchedSources) == 0 {
		utils.Die("Could not match '%s' for module '%s'", arg, ctx.ModuleName())
	}

	return strings.Join(matchedSources, " ")
}

// Ensure that every non-compiled source has been used by at least one
// {{match_srcs}} instance.
func verifyMatchSources(ctx blueprint.BaseModuleContext, matchedNonCompiledSources map[string]bool) {
	// TODO: For modules without any {{match_srcs}} statements, this can produce a red herring since all non-compile
	// sources will be added to implicit deps. Issue a warning to flag this but ignore. In future, fix this
	// to **only** apply when {{match_srcs}} is used.

	unmatchedCount := 0
	for _, matched := range matchedNonCompiledSources {
		if !matched {
			unmatchedCount += 1
		}
	}

	if unmatchedCount > 0 {
		GetLogger().Warn(warnings.UnmatchedNonCompileSrcsWarning, ctx.BlueprintsFile(), ctx.ModuleName())
	}
}

// If the flag is supported by any of the input languages return it,
// otherwise return "" to exclude it
func checkCompilerFlag(flag string, languages []string, tc toolchain.Toolchain) string {
	for _, lang := range languages {
		if tc.CheckFlagIsSupported(lang, flag) {
			return flag
		}
	}
	return ""
}

// Handle {{add_if_supported}}. It checks the compiler flag passed
// on the input and keeps it *if* the compiler supports it.
func setupAddIfSupported(ctx blueprint.BaseModuleContext,
	propfnmap map[string]template.FuncMap) {

	if t, ok := ctx.Module().(moduleWithBuildProps); ok {
		build := t.build()
		tc := backend.Get().GetToolchain(build.TargetType)

		addtoFuncmap(propfnmap, []string{"Cflags", "Export_cflags"}, "add_if_supported",
			func(s string) string {
				return checkCompilerFlag(s, []string{"c++", "c"}, tc)
			})
		addtoFuncmap(propfnmap, []string{"Cxxflags"}, "add_if_supported",
			func(s string) string {
				return checkCompilerFlag(s, []string{"c++"}, tc)
			})
		addtoFuncmap(propfnmap, []string{"Conlyflags"}, "add_if_supported",
			func(s string) string {
				return checkCompilerFlag(s, []string{"c"}, tc)
			})
	}
}

// Applies late templates to the given module
func applyLateTemplates(ctx blueprint.BaseModuleContext) {

	m, ok := ctx.Module().(Featurable)
	if !ok {
		// Features and templates not supported by this module type
		return
	}

	propfnmap := make(map[string]template.FuncMap)

	// Set up {{match_srcs}} and {{add_if_supported}} handling
	nonCompiledSources := setupMatchSources(ctx, propfnmap)
	setupAddIfSupported(ctx, propfnmap)

	// Add more late templates above this line

	if len(propfnmap) == 0 {
		// If propfnmap is empty, then no late templates are
		// applicable to this module type
		return
	}

	// Generic template expansion
	for _, p := range m.FeaturableProperties() {
		propsVal := reflect.Indirect(reflect.ValueOf(p))

		// Properties have already been expanded, so set stringvalues to nil
		applyLateTemplateRecursive(propsVal, nil, propfnmap)
	}

	verifyMatchSources(ctx, nonCompiledSources)
}

// This mutator handles late templates
//
// These templates have access to more information that normal
// templates in template.go
func lateTemplateMutator(ctx blueprint.TopDownMutatorContext) {
	module := ctx.Module()

	if e, ok := module.(enableable); ok {
		if isEnabled(e) {
			applyLateTemplates(ctx)
		}
	}
}
