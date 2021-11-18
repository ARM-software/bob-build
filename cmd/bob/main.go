/*
 * Copyright 2018, 2021 Arm Limited.
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

package main

import (
	"flag"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/ARM-software/bob-build/core"
	"github.com/ARM-software/bob-build/internal/utils"
)

func main() {
	// The primary builder should use the global flag set because the
	// bootstrap package registers its own flags there.
	flag.Parse()

	cpuprofile, present := os.LookupEnv("BOB_CPUPROFILE")
	if present && cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			utils.Die("%v", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	core.Main()

	memprofile, present := os.LookupEnv("BOB_MEMPROFILE")
	if present && memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			utils.Die("%v", err)
		}
		runtime.GC()
		pprof.WriteHeapProfile(f)
	}
}
