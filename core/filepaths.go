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

// Array of files as a helper for struct attribute collections
// TODO: add the possibility to tag a group of files.
type FilePaths []FilePath

func (fps FilePaths) Contains(query FilePath) bool {
	for _, fp := range fps {
		if fp == query {
			return true
		}
	}
	return false
}

func (fps FilePaths) AppendIfUnique(fp FilePath) FilePaths {
	if !fps.Contains(fp) {
		return append(fps, fp)
	}
	return fps
}

func (fps FilePaths) Merge(other FilePaths) FilePaths {
	return append(fps, other...)
}

func (fps FilePaths) Iterate() <-chan FilePath {
	c := make(chan FilePath)
	go func() {
		for _, fp := range fps {
			c <- fp
		}
		close(c)
	}()
	return c
}

func (fps FilePaths) IteratePredicate(predicate func(FilePath) bool) <-chan FilePath {
	c := make(chan FilePath)
	go func() {
		for _, fp := range fps {
			if predicate(fp) {
				c <- fp
			}
		}
		close(c)
	}()
	return c
}

func (fps FilePaths) ForEach(functor func(FilePath) bool) {
	for fp := range fps.Iterate() {
		if !functor(fp) {
			break
		}
	}
}

func (fps FilePaths) ForEachIf(predicate func(FilePath) bool, functor func(FilePath) bool) {
	for fp := range fps.IteratePredicate(predicate) {
		if !functor(fp) {
			break
		}
	}
}
