package plugin

import "github.com/bazelbuild/bazel-gazelle/label"

type Registrable interface {
	getName() string
	getRelativePath() string // from bob root
	getLabel() label.Label
}

type Registry struct {
	globalNameMap map[string]Registrable
	labelMap      map[label.Label]Registrable
	pathMap       map[string][]Registrable
}

func (r *Registry) register(m Registrable) {
	r.globalNameMap[m.getName()] = m
	r.pathMap[m.getRelativePath()] = append(r.pathMap[m.getRelativePath()], m)
	r.labelMap[m.getLabel()] = m
}

func (r *Registry) nameExists(name string) bool {
	_, ok := r.globalNameMap[name]
	return ok
}

func (r *Registry) labelExists(l label.Label) bool {
	_, ok := r.labelMap[l]
	return ok
}

func (r *Registry) retrieveByName(name string) (reg Registrable, ok bool) {
	reg, ok = r.globalNameMap[name]
	return reg, ok
}

func (r *Registry) retrieveByLabel(l label.Label) (reg Registrable, ok bool) {
	reg, ok = r.labelMap[l]
	return reg, ok
}

func (r *Registry) retrieveByPath(path string) (regs []Registrable, ok bool) {
	regs, ok = r.pathMap[path]
	return regs, ok
}

func NewRegistry() *Registry {
	return &Registry{
		globalNameMap: map[string]Registrable{},
		labelMap:      map[label.Label]Registrable{},
		pathMap:       map[string][]Registrable{},
	}
}
