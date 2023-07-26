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
