package kinds

import (
	"github.com/bazelbuild/bazel-gazelle/rule"
)

var Kinds = map[string]rule.KindInfo{
	"filegroup": {
		NonEmptyAttrs:  map[string]bool{"srcs": true},
		MergeableAttrs: map[string]bool{"srcs": true},
	},
	"bool_flag": {
		NonEmptyAttrs: map[string]bool{
			"name":                  true,
			"build_setting_default": true,
		},
		MergeableAttrs: map[string]bool{"build_setting_default": true},
	},
	"string_flag": {
		NonEmptyAttrs: map[string]bool{
			"name":                  true,
			"build_setting_default": true,
		},
		MergeableAttrs: map[string]bool{"build_setting_default": true},
	},
	"int_flag": {
		NonEmptyAttrs: map[string]bool{
			"name":                  true,
			"build_setting_default": true,
		},
		MergeableAttrs: map[string]bool{"build_setting_default": true},
	},
	"config_setting": {
		NonEmptyAttrs:  map[string]bool{"name": true},
		MergeableAttrs: map[string]bool{"flag_values": false},
	},
	"selects.config_setting_group": {
		NonEmptyAttrs: map[string]bool{"name": true},
		MergeableAttrs: map[string]bool{
			"match_all": true,
			"match_any": true,
		},
	},
	"cc_library": {
		NonEmptyAttrs: map[string]bool{
			"srcs": true,
			"hdrs": true,
			"deps": true,
		},
		MergeableAttrs: map[string]bool{
			"srcs":          true,
			"hdrs":          true,
			"local_defines": true,
			"defines":       true,
			"copts":         true,
			"alwayslink":    true,
			"linkstatic":    true,
		},
		SubstituteAttrs: map[string]bool{
			"srcs": true,
			"hdrs": true,
			"deps": true,
		},
		ResolveAttrs: map[string]bool{
			"deps": true,
		},
	},
}

var Loads = []rule.LoadInfo{
	{
		Name: "@bazel_skylib//rules:common_settings.bzl",
		Symbols: []string{
			"bool_flag",
			"string_flag",
			"int_flag",
		},
	},
	{
		Name: "@bazel_skylib//lib:selects.bzl",
		Symbols: []string{
			"selects",
		},
	},
}
