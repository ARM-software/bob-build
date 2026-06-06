package flag

import (
	"reflect"
	"sync"

	"github.com/google/blueprint"
)

type FlagParserTableEntry struct {
	PropertyName string
	Tag          Type
	Factory      func(string, blueprint.Module, Type) Flag
}

type FlagParserTable []FlagParserTableEntry

var fieldIndexCache = struct {
	sync.RWMutex
	indexes map[fieldIndexCacheKey][]int
}{
	indexes: make(map[fieldIndexCacheKey][]int),
}

type fieldIndexCacheKey struct {
	typ  reflect.Type
	name string
}

func concreteValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func cachedFieldByName(v reflect.Value, name string) reflect.Value {
	v = concreteValue(v)
	key := fieldIndexCacheKey{
		typ:  v.Type(),
		name: name,
	}

	fieldIndexCache.RLock()
	index, ok := fieldIndexCache.indexes[key]
	fieldIndexCache.RUnlock()
	if !ok {
		field, found := key.typ.FieldByName(name)
		if !found {
			panic("missing flag property field: " + name)
		}
		index = append([]int(nil), field.Index...)

		fieldIndexCache.Lock()
		if cached, ok := fieldIndexCache.indexes[key]; ok {
			index = cached
		} else {
			fieldIndexCache.indexes[key] = index
		}
		fieldIndexCache.Unlock()
	}

	return v.FieldByIndex(index)
}

// Helper method to scrape many properties from a module struct.
func ParseFromProperties(owner blueprint.Module, luts FlagParserTable, s interface{}) (ret Flags) {
	v := reflect.ValueOf(s)
	for _, entry := range luts {
		values := cachedFieldByName(v, entry.PropertyName)
		for i := 0; i < values.Len(); i++ {
			ret = append(ret, entry.Factory(values.Index(i).String(), owner, entry.Tag))
		}
	}
	return
}
