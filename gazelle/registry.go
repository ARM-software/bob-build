package plugin

import "github.com/bazelbuild/bazel-gazelle/label"

type Registrable interface {
	getName() string
	getRelativePath() string // from bob root
	getLabel() label.Label
}

type Registry struct {
	globalNameMap map[string]*Registrable
	labelMap      map[label.Label]*Registrable
	pathMap       map[string][]*Registrable
}

func (r *Registry) register(m Registrable) {
	r.globalNameMap[m.getName()] = &m
	r.pathMap[m.getRelativePath()] = append(r.pathMap[m.getRelativePath()], &m)
	r.labelMap[m.getLabel()] = &m
}

func (r *Registry) nameExists(name string) bool {
	return r.globalNameMap[name] != nil
}

func (r *Registry) labelExists(l label.Label) bool {
	return r.labelMap[l] != nil
}

func (r *Registry) retrieveByName(name string) *Registrable {
	return r.globalNameMap[name]
}

func (r *Registry) retrieveByLabel(l label.Label) *Registrable {
	return r.labelMap[l]
}

func (r *Registry) retrieveByPath(path string) []*Registrable {
	return r.pathMap[path]
}

func NewRegistry() *Registry {
	return &Registry{
		globalNameMap: map[string]*Registrable{},
		labelMap:      map[label.Label]*Registrable{},
		pathMap:       map[string][]*Registrable{},
	}
}
