package plugin

import (
	"log"

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
	log.Printf("GenerateRules() - NOT IMPLEMENTED\n")
	return language.GenerateResult{}
}
