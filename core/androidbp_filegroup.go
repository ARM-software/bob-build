/*
 * Copyright 2022 Arm Limited.
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
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

func (g *androidBpGenerator) filegroupActions(fg *filegroup, ctx blueprint.ModuleContext) {
	m, err := AndroidBpFile().NewModule("filegroup", fg.shortName())
	if err != nil {
		utils.Die("%v", err.Error())
	}
	var filegroupSrcs []string
	for _, filegroupName := range fg.Properties.Filegroup_srcs {
		filegroupSrcs = append(filegroupSrcs, ":"+filegroupName)
	}

	m.AddStringList("srcs", append(fg.Properties.Srcs, filegroupSrcs...))
}