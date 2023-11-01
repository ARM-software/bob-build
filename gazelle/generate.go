package plugin

import (
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ARM-software/bob-build/gazelle/common"
	"github.com/ARM-software/bob-build/gazelle/registry"
	"github.com/ARM-software/bob-build/gazelle/types"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
)

type MatchType int8

const (
	MatchAll MatchType = iota
	MatchAny
)

const (
	ruleSelectsBzl string = "@bazel_skylib//lib:selects.bzl"
)

var (
	MatchString map[string]MatchType = map[string]MatchType{
		"and": MatchAll,
		"or":  MatchAny,
	}
	OpString map[string]string = map[string]string{
		"and": "&&",
		"or":  "||",
	}
)

// GenerateRules extracts build metadata from source files in a directory.
// GenerateRules is called in each directory where an update is requested
// in depth-first post-order.
//
// args contains the arguments for GenerateRules. This is passed as a
// struct to avoid breaking implementations in the future when new
// fields are added.
//
// A GenerateResult struct is returned. Optional fields may be added to this
// type in the future.
//
// Any non-fatal errors this function encounters should be logged using
// log.Print.
func (e *BobExtension) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	result := language.GenerateResult{}
	rel := filepath.Clean(args.Rel)
	rules := generateConfigs(e.registry, rel)

	if regs, ok := e.registry.RetrieveByPath(rel); ok {
		// To properly test generation of multiple modules
		// at once the order needs to be preserved
		// Sort modules by its `idx` index

		modulesToGen := make([]*Module, 0)

		for _, reg := range regs {
			if mod, ok := reg.(*Module); ok {
				modulesToGen = append(modulesToGen, mod)
			}
		}

		sort.Slice(modulesToGen, func(i, j int) bool {
			return (*modulesToGen[i]).idx < (*modulesToGen[j]).idx
		})

		for _, mod := range modulesToGen {
			m, _ := e.registry.RetrieveByName(mod.GetName())

			if g, ok := m.(types.Generator); ok {

				rule, err := g.GenerateRule()

				if err != nil {
					log.Println(err.Error())
				} else {
					rules = append(rules, rule)
				}
			}
		}
	}

	for _, r := range rules {
		if r.IsEmpty(bobKinds[r.Kind()]) {
			result.Empty = append(result.Empty, r)
		} else {
			result.Gen = append(result.Gen, r)
			result.Imports = append(result.Imports, r.PrivateAttr(""))
		}
	}

	return result
}

func generateConfigs(r *registry.Registry, relPath string) []*rule.Rule {
	rules := make([]*rule.Rule, 0)

	if r != nil {
		// To properly test generation of multiple configs
		// at once the order needs to be preserved
		// Sort all configs by its Position
		configsToGen := make([]*configData, 0)

		regs, ok := r.RetrieveByPath(relPath)
		if !ok {
			return rules
		}

		for _, reg := range regs {
			if cfg, ok := reg.(*configData); ok && cfg.Ignore != "y" {
				configsToGen = append(configsToGen, cfg)
			}
		}

		sort.Slice(configsToGen, func(i, j int) bool {
			return configsToGen[i].Position < configsToGen[j].Position
		})

		for _, v := range configsToGen {

			// v.Datatype should be one of ["bool", "string", "int"]
			if !utils.Contains([]string{"bool", "string", "int"}, v.Datatype) {
				log.Printf("Unsupported config of type '%s'", v.Datatype)
				break
			}

			ruleName := fmt.Sprintf("%s_flag", v.Datatype)
			rFlag := generateFlag(ruleName, v.Name)

			// 'build_setting_default' value is mandatory
			if d, ok := v.Default.([]interface{}); ok && len(d) == 2 {
				setBuildSettingDefault(rFlag, d[1])
			} else {
				// TODO: handle conditional defaults
				switch v.Datatype {
				case "bool":
					setBuildSettingDefault(rFlag, false)
				case "int":
					setBuildSettingDefault(rFlag, 0)
				case "string":
					setBuildSettingDefault(rFlag, "")
				}
			}

			rules = append(rules, rFlag)

			// features are only when (v.Datatype == "bool")
			if v.Datatype == "bool" {
				rConfigSetting := generateConfigSetting(v.Name, v.Depends != nil)
				setConfigSettingFlagValues(rConfigSetting, v.Name, true)
				rules = append(rules, rConfigSetting)

				if v.Depends != nil {
					r, comment := generateDependencies(v.Name, v.Depends, r)
					rules = append(rules, r...)
					rFlag.AddComment("# depends on: " + comment)
				}
			}
		}
	}

	return rules
}

func generateDependencies(name string, value interface{}, r *registry.Registry) ([]*rule.Rule, string) {

	// idx = 0, helper index to start enumerating from
	// for interim rules of `depends on` equation
	rules, last, comment := generateDependenciesInner(name, value, r, 0)

	ruleName := fmt.Sprintf("config_%s", name)
	configRuleName := fmt.Sprintf(":interim_config_%s", name)

	// generate final config's `selects.config_setting_group`
	// for its `depends on` property
	cgRules := generateConfigGroup(ruleName, MatchAll, []string{configRuleName, last}, []string{":__subpackages__"})

	rules = append(rules, cgRules)

	return rules, comment
}

func generateDependenciesInner(name string, value interface{}, r *registry.Registry, idx int64) ([]*rule.Rule, string, string) {
	var comment string
	var ruleLabel string
	rules := make([]*rule.Rule, 0)

	v := reflect.ValueOf(value)

	if v.Kind() == reflect.Slice {

		t := fmt.Sprint(v.Index(0).Interface())

		switch t {
		case "identifier":
			depName := strings.ToLower(fmt.Sprintf("%s", v.Index(1).Interface()))
			r, ok := r.RetrieveByName(depName)

			if !ok {
				log.Printf("Could not retrieve `Registrable` for '%s' name", depName)
			}

			l := r.GetLabel()
			comment = l.String()
			// `selects.config_setting_group` needs a flag's
			// `config_setting` but not flag itself.
			// Comment stays with flag's label for a better readability.
			l.Name = "config_" + l.Name
			ruleLabel = l.String()
		case "and", "or":
			r1, n1, s1 := generateDependenciesInner(name, v.Index(1).Interface(), r, idx+1)
			r2, n2, s2 := generateDependenciesInner(name, v.Index(2).Interface(), r, idx+1)

			rules = append(rules, r1...)
			rules = append(rules, r2...)

			ruleName := fmt.Sprintf("%s_%s_%d", name, t, idx)

			r := generateConfigGroup(ruleName, MatchString[t], []string{n1, n2}, []string{"//visibility:private"})
			r.AddComment("# autogenerated for internal use only")
			rules = append(rules, r)

			// rule label has to be prefixed with ':'
			ruleLabel = fmt.Sprintf(":%s", ruleName)

			comment = fmt.Sprintf("[%s %s %s]", s1, OpString[t], s2)
		default:
			log.Printf("unsupported %s\n", t)
		}
	}

	return rules, ruleLabel, comment
}

func generateConfigGroup(name string, t MatchType, l []string, visibility []string) *rule.Rule {

	r := rule.NewRule("selects.config_setting_group", name)
	r.SetKind("selects.config_setting_group")

	switch t {
	case MatchAll:
		r.SetAttr("match_all", l)
	case MatchAny:
		r.SetAttr("match_any", l)
	}

	if len(visibility) > 0 {
		r.SetAttr("visibility", visibility)
	}

	return r
}

func generateFlag(flagType string, name string) *rule.Rule {

	r := rule.NewRule(flagType, name)
	r.SetKind(flagType)

	return r
}

func setBuildSettingDefault(r *rule.Rule, value interface{}) {

	switch value.(type) {
	// json.Unmarshal grabs numbers as floats
	case float64, float32:
		v := int(reflect.ValueOf(value).Float())
		r.SetAttr("build_setting_default", rule.ExprFromValue(v))
	default:
		r.SetAttr("build_setting_default", rule.ExprFromValue(value))
	}
}

func generateConfigSetting(name string, isDependent bool) *rule.Rule {

	var ruleName string

	if isDependent {
		ruleName = fmt.Sprintf("interim_config_%s", strings.ToLower(name))
	} else {
		ruleName = fmt.Sprintf("config_%s", strings.ToLower(name))
	}

	r := rule.NewRule("config_setting", ruleName)
	r.SetKind("config_setting")

	return r
}

func setConfigSettingFlagValues(r *rule.Rule, name string, value interface{}) {
	if v, ok := getValueString(value); ok {
		arg := &bzl.KeyValueExpr{
			Key:   &bzl.StringExpr{Value: fmt.Sprintf(":%s", name)},
			Value: &bzl.StringExpr{Value: v},
		}
		expr := &bzl.DictExpr{List: []*bzl.KeyValueExpr{arg}, ForceMultiLine: true}

		r.SetAttr("flag_values", expr)
	}
}

func getValueString(value interface{}) (string, bool) {

	switch value.(type) {
	case bool:
		b, _ := value.(bool)
		return strconv.FormatBool(b), true
	case string:
		s, _ := value.(string)
		return s, true
	}

	return "", false
}

// TODO: resolve feature names properly depending on
// the location in `build.bp`
func getFeatureCondition(f string) string {

	if f == common.ConditionDefault {
		return f
	} else {
		return fmt.Sprintf(":config_%s", strings.ToLower(f))
	}
}
