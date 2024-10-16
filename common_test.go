package envio

import (
	"errors"
	"math/bits"
	"reflect"
	"testing"
)

func equal(t *testing.T, exp, got interface{}) {
	if !reflect.DeepEqual(exp, got) {
		t.Fatalf("Not equal:\nexp: %v\ngot: %v", exp, got)
	}
}

func Test_bitSize(t *testing.T) {
	var tests = []struct {
		name string

		reflectKind reflect.Kind
		expect      int
	}{
		{
			name:        "bool",
			reflectKind: reflect.Bool,
			expect:      0,
		},
		{
			name:        "int8",
			reflectKind: reflect.Int8,
			expect:      8,
		},
		{
			name:        "int16",
			reflectKind: reflect.Int16,
			expect:      16,
		},
		{
			name:        "int32",
			reflectKind: reflect.Int32,
			expect:      32,
		},
		{
			name:        "int64",
			reflectKind: reflect.Int64,
			expect:      64,
		},
		{
			name:        "uint8",
			reflectKind: reflect.Uint8,
			expect:      8,
		},
		{
			name:        "uint16",
			reflectKind: reflect.Uint16,
			expect:      16,
		},
		{
			name:        "uint32",
			reflectKind: reflect.Uint32,
			expect:      32,
		},
		{
			name:        "uint64",
			reflectKind: reflect.Uint64,
			expect:      64,
		},
		{
			name:        "int",
			reflectKind: reflect.Int,
			expect:      bits.UintSize,
		},
		{
			name:        "uint",
			reflectKind: reflect.Uint,
			expect:      bits.UintSize,
		},
		{
			name:        "uintptr",
			reflectKind: reflect.Uintptr,
			expect:      bits.UintSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal(t, tt.expect, bitSize(tt.reflectKind))
		})
	}
}

func Test_isEmptyValue(t *testing.T) {
	var a any
	a = 77
	var b any

	var tests = []struct {
		name   string
		value  any
		expect bool
	}{
		{
			name:   "no empty bool",
			value:  true,
			expect: false,
		},
		{
			name:   "empty bool",
			value:  false,
			expect: true,
		},
		{
			name:   "no empty int",
			value:  1,
			expect: false,
		},
		{
			name:   "empty int",
			value:  0,
			expect: true,
		},
		{
			name:   "np empty float",
			value:  1.1,
			expect: false,
		},
		{
			name:   "empty float",
			value:  0.0,
			expect: true,
		},
		{
			name:   "no empty string",
			value:  "a",
			expect: false,
		},
		{
			name:   "empty string",
			value:  "",
			expect: true,
		},
		{
			name:   "no empty pointer",
			value:  &struct{}{},
			expect: false,
		},
		{
			name:   "empty pointer",
			value:  (*struct{})(nil),
			expect: true,
		},
		{
			name:   "no empty slice",
			value:  []int{1},
			expect: false,
		},
		{
			name:   "empty slice",
			value:  []int{},
			expect: true,
		},
		{
			name:   "no empty interface",
			value:  a,
			expect: false,
		},
		{
			name:   "empty interface",
			value:  b,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal(t, tt.expect, isEmptyValue(reflect.ValueOf(tt.value)))
		})
	}
}

func Test_contextSetError(t *testing.T) {
	tagName := "tagName"
	str := "set/get"

	var tests = []struct {
		name   string
		ctx    context
		expect error
	}{
		{
			name: "error for structs",
			ctx: context{
				structName: "structName",
				field: &field{
					name: "fieldName",
					typ:  reflect.TypeOf(true),
				},
				err: ErrNotSupportType,
			},
			expect: errors.New("tagName: cannot set/get Go struct field structName.fieldName of type bool: cannot support type"),
		},
		{
			name: "error for simple types",
			ctx: context{
				structName: "",
				field: &field{
					typ: reflect.TypeOf(true),
				},
				err: ErrNotSupportType,
			},
			expect: errors.New("tagName: cannot set/get Go value of type bool: cannot support type"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ctx.setError(tagName, str, tt.ctx.err)
			if tt.expect != nil {
				equal(t, tt.expect.Error(), tt.ctx.err.Error())
				return
			}
			equal(t, nil, tt.ctx.err)
		})

	}
}
