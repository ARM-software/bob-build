/*
 * Copyright 2023 Arm Limited.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

func (b *Build) getBuildWrapperAndDeps(ctx blueprint.ModuleContext) (string, []string) {
	if b.Build_wrapper != nil {
		depargs := map[string]string{}
		files := getDependentArgsAndFiles(ctx, depargs)

		// Replace any property usage in buildWrapper
		buildWrapper := *b.Build_wrapper
		for k, v := range depargs {
			buildWrapper = strings.Replace(buildWrapper, "${"+k+"}", v, -1)
		}

		return buildWrapper, files
	}

	return "", []string{}
}

func (b *Build) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	b.BuildProps.processPaths(ctx, g)
	b.CommonProps.processPaths(ctx, g)
}
