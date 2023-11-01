package registry

import (
	"github.com/bazelbuild/bazel-gazelle/label"
)

type Registrable interface {
	GetName() string
	GetRelativePath() string // from bob root
	GetLabel() label.Label
}

type Registry struct {
	globalNameMap map[string]Registrable
	labelMap      map[label.Label]Registrable
	pathMap       map[string][]Registrable
}

func (r *Registry) Register(m Registrable) {
	r.globalNameMap[m.GetName()] = m
	r.pathMap[m.GetRelativePath()] = append(r.pathMap[m.GetRelativePath()], m)
	r.labelMap[m.GetLabel()] = m
}

func (r *Registry) NameExists(name string) bool {
	_, ok := r.globalNameMap[name]
	return ok
}

func (r *Registry) LabelExists(l label.Label) bool {
	_, ok := r.labelMap[l]
	return ok
}

func (r *Registry) RetrieveByName(name string) (reg Registrable, ok bool) {
	reg, ok = r.globalNameMap[name]
	return reg, ok
}

func (r *Registry) RetrieveByLabel(l label.Label) (reg Registrable, ok bool) {
	reg, ok = r.labelMap[l]
	return reg, ok
}

func (r *Registry) RetrieveByPath(path string) (regs []Registrable, ok bool) {
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
