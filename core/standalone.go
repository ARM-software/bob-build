/*
 * Copyright 2018-2019 Arm Limited.
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

/*
 * This file is included when Bob is being run as a standalone binary, i.e. for
 * the Ninja and Android Make generators.
 */

package core

import (
	"path/filepath"

	"github.com/google/blueprint"
)

var (
	jsonPath = filepath.Join(builddir, "config.json")
)

type moduleBase struct {
	blueprint.SimpleName
}

// configProvider allows the retrieval of configuration
type configProvider interface {
	Config() interface{}
}

func getConfig(ctx configProvider) *bobConfig {
	return ctx.Config().(*bobConfig)
}
