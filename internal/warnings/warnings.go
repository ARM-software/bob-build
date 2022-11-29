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

package warnings

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type Category string

const (
	DefaultSrcsWarning    Category = "DefaultSrcsWarning"
	DeprecationWarning    Category = "DeprecationWarning"
	DirectPathsWarning    Category = "DirectPathsWarning"
	GenerateRuleWarning   Category = "GenerateRuleWarning"
	PropertyWarning       Category = "PropertyWarning"
	RelativeUpLinkWarning Category = "RelativeUpLinkWarning"
	UserWarning           Category = "UserWarning"
)

var categoriesMap = map[string]Category{
	string(DefaultSrcsWarning):    DefaultSrcsWarning,
	string(DeprecationWarning):    DeprecationWarning,
	string(DirectPathsWarning):    DirectPathsWarning,
	string(GenerateRuleWarning):   GenerateRuleWarning,
	string(PropertyWarning):       PropertyWarning,
	string(RelativeUpLinkWarning): RelativeUpLinkWarning,
	string(UserWarning):           UserWarning,
}

type Action string

const (
	IgnoreAction  Action = "ignore"
	WarningAction Action = "warning"
	ErrorAction   Action = "error"
)

var actionsMap = map[string]Action{
	"I": IgnoreAction,
	"W": WarningAction,
	"E": ErrorAction,
}

type WarningLogger struct {
	out          *csv.Writer
	mu           sync.Mutex
	filters      map[Category]Action
	globalAction Action
	errors       int
}

func New(out io.Writer, filters string) *WarningLogger {
	w := csv.NewWriter(out)
	w.Write([]string{"BpFile", "BpModule", "WarningAction", "WarningMessage", "WarningCategory"})
	w.Flush()

	f, g := parseFilters(filters)

	return &WarningLogger{out: w, filters: f, globalAction: g}
}

func parseFilters(f string) (filters map[Category]Action, globalAction Action) {

	filters = make(map[Category]Action)

	fn := func(c rune) bool {
		return c == ' '
	}

	if f != "" {
		for _, subFilter := range strings.FieldsFunc(f, fn) {
			parts := strings.SplitN(subFilter, ":", 2)

			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "Wrong warnings filter expression '%s'\n", subFilter)
				continue
			}

			c, a := parts[0], parts[1]

			if _, ok := actionsMap[a]; !ok {
				fmt.Fprintf(os.Stderr, "Wrong filter action '%s'\n", subFilter)
				continue
			}

			if c == "*" {
				if globalAction != "" {
					fmt.Fprintf(os.Stderr, "Overriding wildcard (*) not allowed: '%s'\n", subFilter)
				} else {
					globalAction = actionsMap[a]
				}

				continue
			}

			if category, ok := categoriesMap[c]; ok {
				if _, ok := filters[category]; ok {
					fmt.Fprintf(os.Stderr, "Overriding warning category not allowed: '%s'\n", subFilter)
					continue
				}

				filters[category] = actionsMap[a]
			} else {
				fmt.Fprintf(os.Stderr, "Wrong filter category '%s'\n", subFilter)
			}
		}
	}

	if globalAction == "" {
		globalAction = IgnoreAction
	}

	return
}

func (w *WarningLogger) ErrorWarnings() int {
	return w.errors
}

func (w *WarningLogger) Warn(category Category, bpFile string, bpModule string, message string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	action, ok := w.filters[category]

	if !ok {
		action = w.globalAction
	}

	if action == ErrorAction {
		w.errors++
	}

	if action != IgnoreAction {
		io.WriteString(os.Stderr, fmt.Sprintf("%s:%s: %s: %s [%s]\n", bpFile, bpModule, action, message, category))
	}

	w.out.Write([]string{bpFile, bpModule, string(action), message, string(category)})
	w.out.Flush()

	return w.out.Error()
}
