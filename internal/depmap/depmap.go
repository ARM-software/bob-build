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

package depmap

import (
	"sync"
)

type Depmap struct {
	store map[string][]string
	lock  sync.RWMutex
}

func (d *Depmap) SetDeps(key string, deps []string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.store[key] = deps
}

func (d *Depmap) AddDeps(key string, deps []string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.store[key] = append(d.store[key], deps...)
}

func (d *Depmap) GetDeps(key string) []string {
	d.lock.Lock()
	defer d.lock.Unlock()

	deps, ok := d.store[key]
	if ok {
		return deps
	} else {
		return []string{}
	}
}

type visitMap map[string]bool

func (d *Depmap) newVisitMap(starting string) visitMap {
	visited := visitMap{}
	for k := range d.store {
		visited[k] = false
	}
	visited[starting] = true
	return visited
}

func (m visitMap) visited(key string) bool {
	visited, exists := m[key]
	return exists && visited
}

func (m visitMap) visit(key string) {
	m[key] = true
}

type CallbackFn func(string)

// Traverses the dependancies, detects loops.
func (d *Depmap) Traverse(key string, visit CallbackFn, loop CallbackFn) {
	d.lock.Lock()
	defer d.lock.Unlock()
	vmap := d.newVisitMap(key)
	for _, dep := range d.store[key] {
		d.traverseRecursive(dep, vmap, visit, loop)
	}
}

func (d *Depmap) traverseRecursive(key string, vmap visitMap, visit CallbackFn, loop CallbackFn) {
	if vmap.visited(key) {
		loop(key)
	} else {
		vmap.visit((key))
		visit(key)
		for _, dep := range d.store[key] {
			d.traverseRecursive(dep, vmap, visit, loop)
		}
	}
}

// Returns all deps, including transitive ones.
// Uses a very simple depth first approach, ignore circular dependencies these should be handled separately.
func (d *Depmap) GetAllDeps(key string) (out []string) {
	d.Traverse(key,
		func(k string) {
			out = append(out, k)
		},
		func(k string) {},
	)

	return
}

func NewDepmap() Depmap {
	return Depmap{
		store: map[string][]string{},
		lock:  sync.RWMutex{},
	}
}
