package envio

import (
	"errors"
	"os"
	"testing"
)

type simple struct {
	A string `env:"ENV_A,m"`
	B bool   `env:"ENV_B"`
	C int    `env:"ENV_C"`
	D float64
	e string
}

type nested struct {
	X string `env:"ENV_X"`
	Y bool   `env:"ENV_Y"`
	Z int    `env:"ENV_Z"`
	simple
}

type skip struct {
	S string `env:"-"`
	K int    `env:"-"`
	*simple
}

type pointer struct {
	A *int
}

type deepEmbed struct {
	Foo nested
	Bar string `env:"BAR"`
}

type slcArr struct {
	Slc      []bool `env:"ENV_SLC"`
	Arr      [5]int `env:"ENV_ARR"`
	Bytes    []byte `env:"ENV_BYTES"`
	BytesRaw []byte `env:"ENV_BYTES_RAW,raw"`
}

type env struct {
	name  string
	value string
}

func Test_Set(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		expect []env
		err    error
	}{
		{
			name:  "all envs",
			input: simple{A: "test", B: true, C: 28, D: 3.14, e: "not exported"},
			expect: []env{
				{name: "ENV_A", value: "test"},
				{name: "ENV_B", value: "true"},
				{name: "ENV_C", value: "28"},
				{name: "D", value: "3.14"},
			},
		},
		{
			name: "all fields with nested",
			input: nested{
				X:      "value",
				Y:      false,
				Z:      111,
				simple: simple{A: "test", B: true, C: 28, D: 3.14, e: "not exported"},
			},
			expect: []env{
				{name: "ENV_A", value: "test"},
				{name: "ENV_B", value: "true"},
				{name: "ENV_C", value: "28"},
				{name: "D", value: "3.14"},
				{name: "ENV_X", value: "value"},
				{name: "ENV_Y", value: "false"},
				{name: "ENV_Z", value: "111"},
			},
		},
		{
			name: "all fields with skip",
			input: &skip{
				S:      "none",
				K:      35,
				simple: &simple{A: "test", B: true, C: 28, D: 3.14, e: "not exported"},
			},
			expect: []env{
				{name: "ENV_A", value: "test"},
				{name: "ENV_B", value: "true"},
				{name: "ENV_C", value: "28"},
				{name: "D", value: "3.14"},
				{name: "S", value: ""},
				{name: "K", value: ""},
			},
		},
		{
			name:  "nil pointer field",
			input: &pointer{},
		},
		{
			name: "deep embedding",
			input: &deepEmbed{
				Foo: nested{
					X:      "value",
					Y:      false,
					Z:      111,
					simple: simple{A: "test", B: true, C: 28, D: 3.14, e: "not exported"},
				},
				Bar: "foo",
			},
			expect: []env{
				{name: "ENV_A", value: "test"},
				{name: "ENV_B", value: "true"},
				{name: "ENV_C", value: "28"},
				{name: "D", value: "3.14"},
				{name: "ENV_X", value: "value"},
				{name: "ENV_Y", value: "false"},
				{name: "ENV_Z", value: "111"},
				{name: "BAR", value: "foo"},
			},
		},
		{
			name: "arrays and slices",
			input: &slcArr{
				Slc:      []bool{true, false, true},
				Arr:      [5]int{0, 5, 8},
				Bytes:    []byte{65, 66, 67, 68},
				BytesRaw: []byte{65, 66, 67, 68},
			},
			expect: []env{
				{name: "ENV_SLC", value: testEnvSlc},
				{name: "ENV_ARR", value: testEnvArr},
				{name: "ENV_BYTES", value: testEnvBytes},
				{name: "ENV_BYTES_RAW", value: "ABCD"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Set(tt.input)
			if tt.err != nil {
				equal(t, tt.err.Error(), err.Error())
				return
			}
			equal(t, nil, err)

			for _, v := range tt.expect {
				equal(t, v.value, os.Getenv(v.name))
			}

			os.Clearenv()
		})
	}
}

func Test_SetNoOverwrite(t *testing.T) {
	equal(t, nil, os.Setenv("ENV_A", "test"))

	s := simple{B: true, C: 28, D: 3.14}

	expect := []env{
		{name: "ENV_A", value: "test"},
		{name: "ENV_B", value: "true"},
		{name: "ENV_C", value: "28"},
		{name: "D", value: "3.14"},
	}

	equal(t, nil, Set(s))

	for _, v := range expect {
		equal(t, v.value, os.Getenv(v.name))
	}

	os.Clearenv()
}

func Test_Get(t *testing.T) {
	a := 28

	tests := []struct {
		name   string
		envs   []env
		input  any
		expect any
		err    error
	}{
		{
			name: "all envs",
			envs: []env{
				{name: "ENV_A", value: "test"},
				{name: "ENV_B", value: "true"},
				{name: "ENV_C", value: "28"},
				{name: "D", value: "3.14"},
			},
			input:  &simple{A: "a", e: "not exported"},
			expect: &simple{A: "test", B: true, C: 28, D: 3.14, e: "not exported"},
		},
		{
			name: "missing mandatory env",
			envs: []env{
				{name: "ENV_B", value: "true"},
				{name: "ENV_C", value: "28"},
				{name: "D", value: "3.14"},
			},
			input: new(simple),
			err:   errors.New("env: the required variable $ENV_A is missing"),
		},
		{
			name: "invalid syntax",
			envs: []env{
				{name: "ENV_A", value: "test"},
				{name: "ENV_B", value: "?"},
			},
			input: new(simple),
			err:   errors.New("env: cannot get data into Go struct field simple.ENV_B of type bool: invalid syntax"),
		},
		{
			name: "all fields with nested",
			envs: []env{
				{name: "ENV_A", value: "test"},
				{name: "ENV_B", value: "true"},
				{name: "ENV_C", value: "28"},
				{name: "D", value: "3.14"},
				{name: "ENV_X", value: "value"},
				{name: "ENV_Y", value: "false"},
				{name: "ENV_Z", value: "111"},
			},
			input: new(nested),
			expect: &nested{
				X:      "value",
				Y:      false,
				Z:      111,
				simple: simple{A: "test", B: true, C: 28, D: 3.14},
			},
		},
		{
			name: "all fields with skip",
			envs: []env{
				{name: "ENV_A", value: "test"},
				{name: "ENV_B", value: "true"},
				{name: "ENV_C", value: "28"},
				{name: "D", value: "3.14"},
				{name: "S", value: "s"},
				{name: "K", value: "7"},
			},
			input: &skip{
				S:      "none",
				K:      35,
				simple: new(simple),
			},
			expect: &skip{
				S:      "none",
				K:      35,
				simple: &simple{A: "test", B: true, C: 28, D: 3.14},
			},
		},
		{
			name: "nil pointer field",
			envs: []env{
				{name: "A", value: "28"},
			},
			input: new(pointer),
			expect: &pointer{
				A: &a,
			},
		},
		{
			name: "deep embedding",
			envs: []env{
				{name: "ENV_A", value: "test"},
				{name: "ENV_B", value: "true"},
				{name: "ENV_C", value: "28"},
				{name: "D", value: "3.14"},
				{name: "ENV_X", value: "value"},
				{name: "ENV_Y", value: "false"},
				{name: "ENV_Z", value: "111"},
				{name: "BAR", value: "foo"},
			},
			input: new(deepEmbed),
			expect: &deepEmbed{
				Foo: nested{
					X:      "value",
					Y:      false,
					Z:      111,
					simple: simple{A: "test", B: true, C: 28, D: 3.14},
				},
				Bar: "foo",
			},
		},
		{
			name: "arrays and slices",
			envs: []env{
				{name: "ENV_SLC", value: testEnvSlc},
				{name: "ENV_ARR", value: testEnvArr},
				{name: "ENV_BYTES", value: testEnvBytes},
				{name: "ENV_BYTES_RAW", value: "ABCD"},
			},
			input: new(slcArr),
			expect: &slcArr{
				Slc:      []bool{true, false, true},
				Arr:      [5]int{0, 5, 8},
				Bytes:    []byte{65, 66, 67, 68},
				BytesRaw: []byte{65, 66, 67, 68},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()

			for _, v := range tt.envs {
				equal(t, nil, os.Setenv(v.name, v.value))
			}

			err := Get(tt.input)
			if tt.err != nil {
				equal(t, tt.err.Error(), err.Error())
				return
			}
			equal(t, nil, err)
			equal(t, tt.expect, tt.input)
		})
	}
}
