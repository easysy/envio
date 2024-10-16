package envio

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"sync"
)

const getError = "get data into"

func (e *engine) get(v any) error {
	if t := reflect.ValueOf(v).Kind(); t != reflect.Pointer {
		return fmt.Errorf("%s: the input value is not a pointer", name)
	}

	s := e.newGetState()
	defer getStatePool.Put(s)

	s.get(v)
	return s.err
}

type getterState struct {
	*engine
	context
	*bytes.Buffer
}

var getStatePool sync.Pool

func (e *engine) newGetState() *getterState {
	if p := getStatePool.Get(); p != nil {
		s := p.(*getterState)
		s.err = nil
		s.Reset()
		return s
	}

	s := &getterState{engine: e, Buffer: new(bytes.Buffer)}
	s.field = new(field)
	return s
}

func (s *getterState) get(v any) {
	if err := s.reflectValue(reflect.ValueOf(v)); err != nil {
		if !errors.Is(err, errExist) {
			s.setError(name, getError, err)
		}
	}
}

func (s *getterState) reflectValue(v reflect.Value) error {
	s.context.field.typ = v.Type()
	return s.cachedFunctions(s.context.field.typ).getterFunc(s, v)
}

func (s *getterState) getEnv() error {
	str := os.Getenv(s.field.name)
	if s.field.mandatory && str == "" {
		s.err = fmt.Errorf("%s: the required variable $%s is missing", name, s.field.name)
		return errExist
	}
	s.WriteString(str)
	return nil
}

type getterFunc func(*getterState, reflect.Value) error

func (f *structFields) get(s *getterState, v reflect.Value) (err error) {
	s.structName = v.Type().Name()

	for _, s.field = range *f {
		s.Reset()
		rv := v.Field(s.field.index)

		if s.field.embedded != nil {
			if rv.Kind() == reflect.Pointer {
				if rv.IsNil() {
					s.err = fmt.Errorf("%s: %w: %s", name, ErrPointerToUnexported, rv.Type().Elem())
					return errExist
				}
				rv = rv.Elem()
			}

			if err = s.field.embedded.get(s, rv); err != nil {
				return
			}
			continue
		}

		if err = s.field.functions.getterFunc(s, rv); err != nil {
			return
		}
	}

	return
}

func boolProc(s string, v reflect.Value) error {
	r, err := strconv.ParseBool(s)
	v.SetBool(r)
	return err
}

func intProc(s string, v reflect.Value) error {
	r, err := strconv.ParseInt(s, 10, bitSize(v.Kind()))
	v.SetInt(r)
	return err
}

func uintProc(s string, v reflect.Value) error {
	r, err := strconv.ParseUint(s, 10, bitSize(v.Kind()))
	v.SetUint(r)
	return err
}

func floatProc(s string, v reflect.Value) error {
	r, err := strconv.ParseFloat(s, bitSize(v.Kind()))
	v.SetFloat(r)
	return err
}

func pointerProc(s string, v reflect.Value) error {
	rv := reflect.New(v.Type().Elem())
	parser := getProc(rv.Type())
	if err := parser(s, rv.Elem()); err != nil {
		return err
	}
	v.Set(rv)
	return nil
}

func stringParser(s string, v reflect.Value) error {
	v.SetString(s)
	return nil
}

func getProc(t reflect.Type) func(string, reflect.Value) error {
	switch t.Elem().Kind() {
	case reflect.Bool:
		return boolProc
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intProc
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintProc
	case reflect.Float32, reflect.Float64:
		return floatProc
	case reflect.Pointer:
		return pointerProc
	case reflect.String:
		return stringParser
	default:
		return nil
	}
}

func getGetter(s *getterState, v reflect.Value) error {
	rv := reflect.New(v.Type())

	f, ok := rv.Interface().(Getter)
	if !ok {
		return nil
	}

	if err := s.getEnv(); err != nil {
		return err
	}

	if err := f.GetENV(slices.Clone(s.Bytes())); err != nil {
		return err
	}

	v.Set(rv.Elem())
	return nil
}

func boolGetter(s *getterState, v reflect.Value) error {
	if err := s.getEnv(); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	return boolProc(s.String(), v)
}

func intGetter(s *getterState, v reflect.Value) error {
	if err := s.getEnv(); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	return intProc(s.String(), v)
}

func uintGetter(s *getterState, v reflect.Value) error {
	if err := s.getEnv(); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	return uintProc(s.String(), v)
}

func floatGetter(s *getterState, v reflect.Value) error {
	if err := s.getEnv(); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	return floatProc(s.String(), v)
}

func arrayGetter(t reflect.Type) getterFunc {
	proc := getProc(t)
	if proc == nil {
		return unsupportedTypeGetter
	}

	return func(s *getterState, v reflect.Value) error {
		if err := s.getEnv(); err != nil {
			return err
		}
		if s.Len() == 0 {
			return nil
		}
		if s.field.raw && v.Type().Elem().Kind() == reflect.Uint8 {
			if s.Len() > v.Len() {
				return errors.New("index out of range")
			}
			for i, b := range s.Bytes() {
				v.Index(i).SetUint(uint64(b))
			}
			return nil
		}
		bs := bytes.Split(s.Bytes(), s.separator)
		if len(bs) > v.Len() {
			return errors.New("index out of range")
		}
		for i, r := range bs {
			if err := proc(string(r), v.Index(i)); err != nil {
				return err
			}
		}
		return nil
	}
}

func interfaceGetter(s *getterState, v reflect.Value) error {
	if v.IsNil() {
		s.err = ErrNilInterface
		return errExist
	}
	return s.reflectValue(v.Elem())
}

func pointerGetter(s *getterState, v reflect.Value) error {
	if v.IsNil() {
		rv := reflect.New(v.Type().Elem())
		if err := s.reflectValue(rv.Elem()); err != nil {
			return err
		}
		if !isEmptyValue(rv.Elem()) {
			v.Set(rv)
		}
		return nil
	}
	return s.reflectValue(v.Elem())
}

func sliceGetter(t reflect.Type) getterFunc {
	parser := getProc(t)
	if parser == nil {
		return unsupportedTypeGetter
	}

	return func(s *getterState, v reflect.Value) error {
		if err := s.getEnv(); err != nil {
			return err
		}
		if s.Len() == 0 {
			return nil
		}
		if s.field.raw && v.Type().Elem().Kind() == reflect.Uint8 {
			v.SetBytes(slices.Clone(s.Bytes()))
			return nil
		}
		bs := bytes.Split(s.Bytes(), s.separator)
		v.Set(reflect.MakeSlice(t, len(bs), len(bs)))
		for i, r := range bs {
			if err := parser(string(r), v.Index(i)); err != nil {
				return err
			}
		}
		return nil
	}
}

func stringGetter(s *getterState, v reflect.Value) error {
	if err := s.getEnv(); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	return stringParser(s.String(), v)
}

func structGetter(s *getterState, v reflect.Value) error {
	f := s.cachedFields(v.Type())
	return f.get(s, v)
}

func unsupportedTypeGetter(s *getterState, _ reflect.Value) error {
	s.err = ErrNotSupportType
	return errExist
}
