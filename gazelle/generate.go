package plugin

import (
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
)

const (
	ruleSelectsBzl string = "@bazel_skylib//lib:selects.bzl"
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
	rules := generateConfigs(e.configs, rel)

	// FIXME: Gazelle does not support load statements when macros are packaged
	// into a struct.
	// Temporarily add load statement for `selects.config_setting_group`
	// when it's generated.
	for _, r := range rules {
		if strings.HasPrefix(r.Kind(), "selects.") {
			if args.File != nil {
				var load *rule.Load
				for _, l := range args.File.Loads {
					if l.Name() == ruleSelectsBzl {
						load = l
						load.Add("selects")
						break
					}
				}

				// Add new `load` statement
				if load == nil {
					load = rule.NewLoad(ruleSelectsBzl)
					load.Add("selects")
					load.Insert(args.File, len(args.File.Loads))
				}
			}

			break
		}
	}

	if modules, ok := e.registry.retrieveByPath(rel); ok {
		// To properly test generation of multiple modules
		// at once the order needs to be preserved
		// TODO: improve sorting
		names := make([]string, len(modules))

		for i, m := range modules {
			names[i] = m.getName()
		}

		sort.Strings(names)

		for _, name := range names {
			m, _ := e.registry.retrieveByName(name)

			if g, ok := m.(generator); ok {

				rule, err := g.generateRule()

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

func generateConfigs(c *map[string]configData, relPath string) []*rule.Rule {

	rules := make([]*rule.Rule, 0)

	if c != nil {
		// To properly test generation of multiple configs
		// at once the order needs to be preserved
		// TODO: improve sorting
		configNames := make([]string, len(*c))
		keys := reflect.ValueOf(*c).MapKeys()

		for i, k := range keys {
			configNames[i] = k.String()
		}

		sort.Strings(configNames)

		for _, config := range configNames {
			v := (*c)[config]

			if v.Ignore != "y" && v.RelPath == relPath {

				// v.Datatype should be one of ["bool", "string", "int"]
				if !utils.Contains([]string{"bool", "string", "int"}, v.Datatype) {
					log.Printf("Unsupported config of type '%s'", v.Datatype)
					break
				}

				ruleName := fmt.Sprintf("%s_flag", v.Datatype)
				rFlag := generateFlag(ruleName, strings.ToLower(config))

				// 'build_setting_default' value is mandatory
				if d, ok := v.Default.([]interface{}); ok && len(d) == 2 {
					setBuildSettingDefault(rFlag, d[1])
				} else if _, ok := v.Condition.([]interface{}); ok {
					// TODO: handle condition
					setBuildSettingDefault(rFlag, false)
				} else {
					setBuildSettingDefault(rFlag, false)

				}

				rules = append(rules, rFlag)

				// features are only when (v.Datatype == "bool")
				if v.Datatype == "bool" {
					rConfigSetting := generateConfigSetting(strings.ToLower(config))
					setConfigSettingFlagValues(rConfigSetting, strings.ToLower(config), true)
					rules = append(rules, rConfigSetting)
				}
			}
		}
	}

	return rules
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

func generateConfigSetting(name string) *rule.Rule {

	r := rule.NewRule("config_setting", fmt.Sprintf("config_%s", strings.ToLower(name)))
	r.SetKind("config_setting")

	return r
}

func setConfigSettingFlagValues(r *rule.Rule, name string, value interface{}) {
	if value, ok := getValueString(value); ok {
		arg := &bzl.KeyValueExpr{
			Key:   &bzl.StringExpr{Value: fmt.Sprintf(":%s", name)},
			Value: &bzl.StringExpr{Value: value},
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

	if f == ConditionDefault {
		return f
	} else {
		return fmt.Sprintf(":config_%s", strings.ToLower(f))
	}
}
