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

import "github.com/google/blueprint"

/*
	We are swapping from bob_generate_source to bob_genrule

bob_genrule is made to be a stricter version that is compatible with Android.
For easiest compatibility, we are using Androids format for genrule.
Some properties in the struct may not be useful, but it is better to expose as many
features as possible rather than too few. Some are commented out as they would take special
implementation for features we do not already have in place.
*/
type AndroidGenerateRuleProps struct {
	Out []string
}

type AndroidGenerateCommonProps struct {
	// See https://ci.android.com/builds/submitted/8928481/linux/latest/view/soong_build.html
	Name                string
	Srcs                []string
	Exclude_srcs        []string
	Cmd                 *string
	Depfile             *bool
	Enabled             *bool
	Export_include_dirs []string
	Tool_files          []string
	Tools               []string
}

type androidGenerateCommon struct {
	moduleBase
	EnableableProps
	simpleOutputProducer
	headerProducer
	Properties struct {
		AndroidGenerateCommonProps
	}
}

// Module implementing getGenerateCommonInterface are able to generate output files
type getAndroidGenerateCommonInterface interface {
	getAndroidGenerateCommon() *androidGenerateCommon
}

func (m *androidGenerateCommon) getAndroidGenerateCommon() *androidGenerateCommon {
	return m
}

func getAndroidGenerateCommon(i interface{}) (*androidGenerateCommon, bool) {
	var gsc *androidGenerateCommon
	gsd, ok := i.(getAndroidGenerateCommonInterface)
	if ok {
		gsc = gsd.getAndroidGenerateCommon()
	}
	return gsc, ok
}

type androidGenerateRule struct {
	androidGenerateCommon
	Properties struct {
		AndroidGenerateRuleProps
	}
}

func (m *androidGenerateRule) shortName() string {
	return m.Name()
}

func (m *androidGenerateRule) getEnableableProps() *EnableableProps {
	return &m.EnableableProps
}

func (m *androidGenerateRule) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		g.androidGenerateRuleActions(m, ctx)
	}
}

func (m androidGenerateRule) GetProperties() interface{} {
	return m.Properties
}

func generateRuleAndroidFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &androidGenerateRule{}

	return module, []interface{}{&module.androidGenerateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
