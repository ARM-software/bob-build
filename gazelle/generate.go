package plugin

import (
	"log"
	"path/filepath"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/language"
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
	rel := filepath.Clean(args.Rel)
	result := language.GenerateResult{}

	modules, ok := e.registry.retrieveByPath(rel)

	if !ok {
		return result
	}

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
				// TODO: temporarily limit to `filegroup` rules
				if rule.IsEmpty(bobKinds[rule.Kind()]) || rule.Kind() != "filegroup" {
					result.Empty = append(result.Empty, rule)
				} else {
					result.Gen = append(result.Gen, rule)
					result.Imports = append(result.Imports, rule.PrivateAttr(""))
				}
			}
		}
	}

	return result
}
