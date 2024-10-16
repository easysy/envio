package envio

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"sync"
)

const setError = "set data from"

func (e *engine) set(v any) error {
	s := e.newSetState()
	defer setStatePool.Put(s)

	s.set(v)
	return s.err
}

type setterState struct {
	*engine
	context
	scratch [64]byte
}

var setStatePool sync.Pool

func (e *engine) newSetState() *setterState {
	if p := setStatePool.Get(); p != nil {
		s := p.(*setterState)
		s.err = nil
		return s
	}

	s := &setterState{engine: e}
	s.field = new(field)
	return s
}

func (s *setterState) set(v any) {
	if err := s.reflectValue(reflect.ValueOf(v)); err != nil {
		if !errors.Is(err, errExist) {
			s.setError(name, setError, err)
		}
	}
}

func (s *setterState) reflectValue(v reflect.Value) error {
	s.context.field.typ = v.Type()
	return s.cachedFunctions(s.context.field.typ).setterFunc(s, v)
}

func (s *setterState) setEnv(v []byte) error {
	return os.Setenv(s.field.name, string(v))
}

type setterFunc func(*setterState, reflect.Value) error

func valueFromPtr(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Pointer {
		return v
	}
	if v.IsNil() {
		v = reflect.New(v.Type().Elem())
	}
	return v.Elem()
}

func (f *structFields) set(s *setterState, v reflect.Value) (err error) {
	s.structName = v.Type().Name()

	for _, s.field = range *f {
		rv := v.Field(s.field.index)

		// If the environment variable is mandatory,
		// then to avoid overwriting the value, ignore the field if it is empty.
		if s.field.mandatory && isEmptyValue(rv) {
			continue
		}

		if s.field.embedded != nil {
			if err = s.field.embedded.set(s, valueFromPtr(rv)); err != nil {
				return
			}
			continue
		}

		if err = s.field.functions.setterFunc(s, rv); err != nil {
			return
		}
	}

	return
}

func setSetter(s *setterState, v reflect.Value) error {
	tmp := reflect.ValueOf(v.Interface())
	v = reflect.New(v.Type())
	v.Elem().Set(tmp)

	f, ok := v.Interface().(Setter)
	if !ok {
		return nil
	}

	p, err := f.SetENV()
	if err != nil {
		return err
	}

	return s.setEnv(p)
}

func boolSetter(s *setterState, v reflect.Value) error {
	return s.setEnv(strconv.AppendBool(s.scratch[:0], v.Bool()))
}

func intSetter(s *setterState, v reflect.Value) error {
	return s.setEnv(strconv.AppendInt(s.scratch[:0], v.Int(), 10))
}

func uintSetter(s *setterState, v reflect.Value) error {
	return s.setEnv(strconv.AppendUint(s.scratch[:0], v.Uint(), 10))
}

func floatSetter(s *setterState, v reflect.Value) error {
	return s.setEnv(strconv.AppendFloat(s.scratch[:0], v.Float(), 'g', -1, bitSize(v.Kind())))
}

func interfaceSetter(s *setterState, v reflect.Value) error {
	if v.IsNil() {
		s.err = ErrNilInterface
		return errExist
	}
	return s.reflectValue(v.Elem())
}

func pointerSetter(s *setterState, v reflect.Value) error {
	return s.reflectValue(valueFromPtr(v))
}

func setProc(t reflect.Type) func(*setterState, reflect.Value) []byte {
	switch t.Elem().Kind() {
	case reflect.Bool:
		return func(s *setterState, v reflect.Value) []byte {
			return strconv.AppendBool(s.scratch[:0], v.Bool())
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(s *setterState, v reflect.Value) []byte {
			return strconv.AppendInt(s.scratch[:0], v.Int(), 10)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return func(s *setterState, v reflect.Value) []byte {
			return strconv.AppendUint(s.scratch[:0], v.Uint(), 10)
		}
	case reflect.Float32, reflect.Float64:
		return func(s *setterState, v reflect.Value) []byte {
			return strconv.AppendFloat(s.scratch[:0], v.Float(), 'g', -1, bitSize(v.Kind()))
		}
	case reflect.Pointer:
		return func(s *setterState, v reflect.Value) []byte {
			proc := setProc(v.Type())
			v = valueFromPtr(v)
			return proc(s, v)
		}
	case reflect.String:
		return func(s *setterState, v reflect.Value) []byte {
			return append(s.scratch[:0], v.String()...)
		}
	default:
		return nil
	}
}

func sliceSetter(t reflect.Type) setterFunc {
	proc := setProc(t)
	if proc == nil {
		return unsupportedTypeSetter
	}

	return func(s *setterState, v reflect.Value) error {
		if s.field.raw && v.Type().Elem().Kind() == reflect.Uint8 {
			buf := make([]byte, 0, v.Len())
			for i := 0; i < v.Len(); i++ {
				buf = append(buf, uint8(v.Index(i).Uint()))
			}
			return s.setEnv(buf)
		}

		buf := make([]byte, 0)
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				buf = append(buf, s.separator...)
			}
			buf = append(buf, proc(s, v.Index(i))...)
		}
		return s.setEnv(buf)
	}
}

func stringSetter(s *setterState, v reflect.Value) error {
	return s.setEnv(append(s.scratch[:0], v.String()...))
}

func structSetter(s *setterState, v reflect.Value) error {
	f := s.cachedFields(v.Type())
	return f.set(s, reflect.ValueOf(v.Interface()))
}

func unsupportedTypeSetter(s *setterState, _ reflect.Value) error {
	s.err = ErrNotSupportType
	return errExist
}
