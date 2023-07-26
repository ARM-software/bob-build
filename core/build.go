package core

import (
	"strings"

	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

// A Build represents the whole tree of properties for a 'library' object,
// including its host and target-specific properties
type Build struct {
	CommonProps
	BuildProps
	Target TargetSpecific
	Host   TargetSpecific
	SplittableProps
}

func (b *Build) getTargetSpecific(tgt toolchain.TgtType) *TargetSpecific {
	if tgt == toolchain.TgtTypeHost {
		return &b.Host
	} else if tgt == toolchain.TgtTypeTarget {
		return &b.Target
	} else {
		utils.Die("Unsupported target type: %s", tgt)
	}
	return nil
}

// These function check the boolean pointers - which are only filled if someone sets them
// If not, the default value is returned

func (b *Build) isHostSupported() bool {
	if b.Host_supported == nil {
		return false
	}
	return *b.Host_supported
}

func (b *Build) isTargetSupported() bool {
	if b.Target_supported == nil {
		return true
	}
	return *b.Target_supported
}

func (b *Build) isForwardingSharedLibrary() bool {
	if b.Forwarding_shlib == nil {
		return false
	}
	return *b.Forwarding_shlib
}

func (b *Build) isRpathWanted() bool {
	if b.Add_lib_dirs_to_rpath == nil {
		return false
	}
	return *b.Add_lib_dirs_to_rpath
}

func (b *Build) GetBuildWrapperAndDeps(ctx blueprint.ModuleContext) (string, []string) {
	if b.Build_wrapper != nil {
		depargs := map[string]string{}
		files, _ := getDependentArgsAndFiles(ctx, depargs)

		// Replace any property usage in buildWrapper
		buildWrapper := *b.Build_wrapper
		for k, v := range depargs {
			buildWrapper = strings.Replace(buildWrapper, "${"+k+"}", v, -1)
		}

		return buildWrapper, files
	}

	return "", []string{}
}

func (b *Build) processPaths(ctx blueprint.BaseModuleContext) {
	b.BuildProps.processPaths(ctx)
	b.CommonProps.processPaths(ctx)
}
