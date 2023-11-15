// Maps Bob targets to Bazel labels and vice-versa.
package mapper

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/bazelbuild/bazel-gazelle/label"
)

type Mapper struct {
	byLabel map[*label.Label]interface{}
	byKey   map[string]*label.Label
	mutex   sync.RWMutex
}

func NewMapper() *Mapper {
	return &Mapper{
		byLabel: map[*label.Label]interface{}{},
		byKey:   map[string]*label.Label{},
	}
}

func (m *Mapper) FromLabel(label *label.Label) interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if val, ok := m.byLabel[label]; ok {
		return val
	}

	return nil
}

func (m *Mapper) FromValue(value interface{}) *label.Label {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	key := valueToKey(value)

	if val, ok := m.byKey[key]; ok {
		return val
	}

	return nil

}

func valueToKey(value interface{}) string {
	if v, ok := value.(string); ok {
		return strings.TrimPrefix(v, ":")
	}

	if v, ok := value.(fmt.Stringer); ok {
		return v.String()
	}

	panic(fmt.Sprintf("Cannot determine map key for %#v", value))
}

// Maps a given Bob target to a Label
func (m *Mapper) Map(label *label.Label, value interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := valueToKey(value)
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.String:
		m.byKey[key] = label
		m.byLabel[label] = strings.TrimPrefix(value.(string), ":")
	default:
		m.byKey[key] = label
		m.byLabel[label] = value
	}
}

func MakeLabel(target string, pkgPath string) *label.Label {
	label, err := label.Parse(fmt.Sprintf("//%s:%s", pkgPath, strings.TrimPrefix(target, ":")))
	if err != nil {
		panic(err)
	}
	return &label
}
