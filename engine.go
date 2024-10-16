package envio

import (
	"reflect"
	"strings"
	"sync"
)

const name = "env"

var e = &engine{
	separator: []byte{envSeparator},
}

// Set sets values from v to environment variables.
// If v is nil, Set returns a setter error.
func Set(v any) error {
	return e.set(v)
}

// Get gets values from environment variables to the value pointed to by v.
// If v is nil or not a pointer, Get returns a getter error.
func Get(v any) error {
	return e.get(v)
}

type engine struct {
	separator []byte
}

type functions struct {
	setterFunc
	getterFunc
}

var functionsCache sync.Map // map[reflect.Type]*functions

// cachedFunctions is like typeFunctions but uses a cache to avoid repeated work.
func (e *engine) cachedFunctions(t reflect.Type) *functions {
	if c, ok := functionsCache.Load(t); ok {
		return c.(*functions)
	}

	c, _ := functionsCache.LoadOrStore(t, e.typeFunctions(t))
	return c.(*functions)
}

// typeFunctions returns functions for a type.
func (e *engine) typeFunctions(t reflect.Type) *functions {
	f := new(functions)
	switch t.Kind() {
	case reflect.Bool:
		f.setterFunc = boolSetter
		f.getterFunc = boolGetter
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f.setterFunc = intSetter
		f.getterFunc = intGetter
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		f.setterFunc = uintSetter
		f.getterFunc = uintGetter
	case reflect.Float32, reflect.Float64:
		f.setterFunc = floatSetter
		f.getterFunc = floatGetter
	case reflect.Array:
		f.setterFunc = sliceSetter(t)
		f.getterFunc = arrayGetter(t)
	case reflect.Interface:
		f.setterFunc = interfaceSetter
		f.getterFunc = interfaceGetter
	case reflect.Pointer:
		f.setterFunc = pointerSetter
		f.getterFunc = pointerGetter
	case reflect.Slice:
		f.setterFunc = sliceSetter(t)
		f.getterFunc = sliceGetter(t)
	case reflect.String:
		f.setterFunc = stringSetter
		f.getterFunc = stringGetter
	case reflect.Struct:
		f.setterFunc = structSetter
		f.getterFunc = structGetter
	default:
		f.setterFunc = unsupportedTypeSetter
		f.getterFunc = unsupportedTypeGetter
	}

	if t.Kind() != reflect.Pointer {
		p := reflect.PointerTo(t)
		if p.Implements(setter) {
			f.setterFunc = setSetter
		}
		if p.Implements(getter) {
			f.getterFunc = getGetter
		}
	}

	return f
}

// field represents a single field found in a struct.
type field struct {
	index     int
	name      string
	typ       reflect.Type
	mandatory bool
	raw       bool
	functions *functions
	embedded  structFields
}

type structFields []*field

var fieldCache sync.Map // map[reflect.Type]structFields

// cachedFields is like typeFields but uses a cache to avoid repeated work.
func (e *engine) cachedFields(t reflect.Type) structFields {
	if c, ok := fieldCache.Load(t); ok {
		return c.(structFields)
	}
	c, _ := fieldCache.LoadOrStore(t, e.typeFields(t))
	return c.(structFields)
}

// typeFields returns a list of fields that the setter/getter should recognize for the given type.
func (e *engine) typeFields(t reflect.Type) structFields {
	fs := make(structFields, 0, t.NumField())

	// Scan type for fields to setting/getting.
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		ft := sf.Type

		f := &field{
			index: i,
			name:  sf.Name,
			typ:   ft,
		}

		if sf.Anonymous {
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}

			// Ignore embedded fields of unexported non-struct types.
			if !sf.IsExported() && ft.Kind() != reflect.Struct {
				continue
			}

			// Do not ignore embedded fields of unexported struct types since they may have exported fields.
			f.embedded = e.cachedFields(ft)

			if f.embedded == nil {
				continue
			}

			fs = append(fs, f)
			continue
		} else if !sf.IsExported() {
			// Ignore unexported non-embedded fields.
			continue
		}

		if tag, ok := sf.Tag.Lookup(name); ok {
			// Ignore the field if the tag has a skip value.
			if tag == "-" {
				continue
			}

			val := strings.Split(tag, ",")

			if len(val) != 0 {
				f.name = val[0]

				for _, v := range val {
					switch v {
					case "m":
						f.mandatory = true
					case "raw":
						f.raw = true
					}
				}
			}
		}

		f.functions = e.cachedFunctions(ft)
		fs = append(fs, f)
	}

	return fs
}
