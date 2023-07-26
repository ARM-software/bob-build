package core

import (
	"fmt"
	"reflect"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
)

// Property concatenation.
//
// Most properties have the behavior that later values override earlier
// values. For example if we passed the compiler "-DVALUE=0
// -DVALUE=1", then the macro VALUE would end up as "1".
//
// A few properties have the opposite behaviour. In particular, since
// include search paths specify a set of directories to look for
// headers, the first directory searched overrides all the others.
//
// Since in Features, Defaults and Targets we copy properties from one
// set to another, we want to be consistent in the way we prepend and
// append arguments so that overrides behave as expected.
//
// Fields in property structs can be tagged with
// `bob:"first_overrides"` to get include search path ordering.
// Otherwise they will get cflag ordering.
//
// The function naming assumes cflag ordering, i.e.
// Append: src cflag properties override dst cflag properties
// Prepend: dst cflag properties override src cflag properties

func orderNormal(property string, dstField, srcField reflect.StructField,
	dstValue, srcValue interface{}) (proptools.Order, error) {
	order := proptools.Append
	if proptools.HasTag(srcField, "bob", "first_overrides") {
		order = proptools.Prepend
	}
	return order, nil
}

func orderReverse(property string, dstField, srcField reflect.StructField,
	dstValue, srcValue interface{}) (proptools.Order, error) {
	order := proptools.Prepend
	if proptools.HasTag(srcField, "bob", "first_overrides") {
		order = proptools.Append
	}
	return order, nil
}

func AppendProperties(dst interface{}, src interface{}) error {
	return proptools.ExtendProperties(dst, src, nil, orderNormal)
}

func AppendMatchingProperties(dst []interface{}, src interface{}) error {
	return proptools.ExtendMatchingProperties(dst, src, nil, orderNormal)
}

func PrependProperties(dst interface{}, src interface{}) error {
	return proptools.ExtendProperties(dst, src, nil, orderReverse)
}

func PrependMatchingProperties(dst []interface{}, src interface{}) error {
	return proptools.ExtendMatchingProperties(dst, src, nil, orderReverse)
}

// Applies default options
func DefaultApplierMutator(ctx blueprint.BottomUpMutatorContext) {
	// The mutator is run bottom up, so modules without dependencies
	// will be processed first.
	//
	// This mutator propagates the properties from the direct default
	// dependencies to the current module.

	// No need to do this on defaults modules, as we've flattened the
	// hierarchy
	_, isDefaults := ctx.Module().(*ModuleDefaults)
	if isDefaults {
		return
	}

	var defaultableProps []interface{}

	if d, ok := ctx.Module().(defaultable); ok {
		defaultableProps = d.defaultableProperties()
	} else {
		// Not defaultable.
		return
	}

	// Accumulate properties from direct dependencies into an empty defaults
	accumulatedDef := ModuleDefaults{}
	accumulatedProps := accumulatedDef.defaultableProperties()
	ctx.VisitDirectDeps(func(dep blueprint.Module) {
		if ctx.OtherModuleDependencyTag(dep) == DefaultTag {
			def, ok := dep.(*ModuleDefaults)
			if !ok {
				utils.Die("module %s in %s's defaults is not a default",
					dep.Name(), ctx.ModuleName())
			}

			// Append defaults at the same level to maintain cflag order
			err := appendDefaults(accumulatedProps, def.defaultableProperties())
			if err != nil {
				if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
					ctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
				} else {
					utils.Die("%s", err)
				}
			}
		}
	})

	// Now apply the defaults to the core module
	// Defaults are more generic, so we prepend to the
	// core module properties.
	//
	// Note: when prepending (pointers to) bools we copy
	// the value if the dst is nil, otherwise the dst
	// value is left alone.
	err := prependDefaults(defaultableProps, accumulatedProps)
	if err != nil {
		if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
			ctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
		} else {
			utils.Die("%s", err)
		}
	}
}

func prependDefaults(dst []interface{}, src []interface{}) error {
	// For every property in the destination module (defaultable),
	// we search for the corresponding property within the available
	// set of properties in the source `bob_defaults` module.
	// To prepend them they need to be of the same type.
	for _, defaultableProp := range dst {
		propertyFound := false
		for _, propToApply := range src {
			if reflect.TypeOf(defaultableProp) == reflect.TypeOf(propToApply) {
				err := PrependProperties(defaultableProp, propToApply)

				if err != nil {
					return err
				}

				propertyFound = true
				break
			}
		}

		if !propertyFound {
			return fmt.Errorf("Property of type '%T' was not found in `bob_defaults`", defaultableProp)
		}
	}

	return nil
}

func appendDefaults(dst []interface{}, src []interface{}) error {
	// For every property in the destination module (defaultable),
	// we search for the corresponding property within the available
	// set of properties in the source `bob_defaults` module.
	// To append them they need to be of the same type.
	for _, defaultableProp := range dst {
		propertyFound := false
		for _, propToApply := range src {
			if reflect.TypeOf(defaultableProp) == reflect.TypeOf(propToApply) {
				err := AppendProperties(defaultableProp, propToApply)

				if err != nil {
					return err
				}

				propertyFound = true
				break
			}
		}

		if !propertyFound {
			return fmt.Errorf("Property of type '%T' was not found in `bob_defaults`", defaultableProp)
		}
	}

	return nil
}

// Modules implementing featurable support the use of features and templates.
type Featurable interface {
	FeaturableProperties() []interface{}
	Features() *Features
}

// Used to map a set of properties to destination properties
type propmap struct {
	dst []interface{}
	src *Features
}

// Applies feature specific properties within each module
func featureApplierMutator(ctx blueprint.TopDownMutatorContext) {
	module := ctx.Module()
	cfg := getConfig(ctx)

	if m, ok := module.(Featurable); ok {
		cfgProps := &cfg.Properties

		// FeatureApplier mutator is run first. We need to flatten the
		// feature specific properties in the core set, and where
		// supported, the host-specific and target-specific set.
		var props = []propmap{{m.FeaturableProperties(), m.Features()}}

		// TemplateApplier mutator is run before TargetApplier, so we
		// need to apply templates with the core set, as well as
		// host-specific and target-specific sets (where applicable).
		templProps := append([]interface{}{}, m.FeaturableProperties()...)

		// Apply features in target-specific properties.
		// This should happen for all modules which support host:{} and target:{}
		if ts, ok := module.(targetSpecificLibrary); ok {
			host := ts.getTargetSpecific(toolchain.TgtTypeHost)
			target := ts.getTargetSpecific(toolchain.TgtTypeTarget)

			var tgtprops = []propmap{
				{[]interface{}{host.getTargetSpecificProps()}, &host.Features},
				{[]interface{}{target.getTargetSpecificProps()}, &target.Features},
			}
			props = append(props, tgtprops...)

			templProps = append(templProps, host.getTargetSpecificProps())
			templProps = append(templProps, target.getTargetSpecificProps())
		}

		for _, prop := range props {
			// Feature specific properties get added after core properties.
			//
			// Note: when appending (pointers to) bools we always override
			// the dst value. i.e. feature-specific value takes precedence.
			err := prop.src.AppendProps(prop.dst, cfgProps)
			if err != nil {
				if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
					ctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
				} else {
					utils.Die("%s", err)
				}
			}
		}

		for _, p := range templProps {
			ApplyTemplate(p, cfgProps)
		}

		// Since now Features are no longer needed.
		// Delete them to reduce memory usage.
		m.Features().DeInit()
	}
}
