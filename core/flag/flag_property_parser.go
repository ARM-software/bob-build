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

package flag

import (
	"reflect"

	"github.com/google/blueprint"
)

type FlagParserTableEntry struct {
	PropertyName string
	Tag          Type
	Factory      func(string, blueprint.Module, Type) Flag
}

type FlagParserTable []FlagParserTableEntry

// Helper method to scrape many properties from a module struct.
func ParseFromProperties(owner blueprint.Module, luts FlagParserTable, s interface{}) (ret Flags) {
	for _, entry := range luts {
		s := reflect.Indirect(reflect.ValueOf(&s))
		v := s.Elem().FieldByName(entry.PropertyName)
		for _, s := range v.Interface().([]string) {
			ret = append(ret, entry.Factory(s, owner, entry.Tag))
		}
	}
	return
}
