package plugin

import (
	pluginConfig "github.com/ARM-software/bob-build/gazelle/config"
	"github.com/ARM-software/bob-build/gazelle/util"
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
	result := language.GenerateResult{}

	cfgs := args.Config.Exts[BobExtensionName].(pluginConfig.ConfigMap)
	pc := cfgs[args.Rel]

	if pc.IsIgnored {
		return result
	}

	for _, file := range args.RegularFiles {
		if util.Contains(pc.Mconfig.Filenames, file) {
			if ast, ok := pc.Files[file]; ok {
				mr := pc.Mconfig.Builder.Build(args, ast)
				result = util.MergeResults(result, mr)
			}
		}
	}

	for _, file := range args.RegularFiles {
		if util.Contains(pc.Blueprint.Filenames, file) {
			if ast, ok := pc.Files[file]; ok {
				br := pc.Blueprint.Builder.Build(args, ast)
				result = util.MergeResults(result, br)
			}
		}
	}

	// Check if there are any rules to be generated from the logical expression module.
	result = util.MergeResults(result, pc.Logic.Builder.Build(args))
	return result
}
