package envio

import (
	"bytes"
	"os"
	"testing"
)

type ai int

func (i *ai) SetENV() ([]byte, error) {
	if *i == 1 {
		return []byte("yes"), nil
	}
	if *i == -1 {
		return []byte("no"), nil
	}
	return []byte("unknown"), nil
}

type assert struct {
	AI ai `env:"AI"`
}

func Test_SetENV(t *testing.T) {
	tests := []struct {
		input  assert
		expect env
	}{
		{
			input:  assert{AI: 0},
			expect: env{name: "AI", value: "unknown"},
		},
		{
			input:  assert{AI: 1},
			expect: env{name: "AI", value: "yes"},
		},
		{
			input:  assert{AI: -1},
			expect: env{name: "AI", value: "no"},
		},
	}

	for _, tt := range tests {
		err := Set(tt.input)
		equal(t, nil, err)
		equal(t, tt.expect.value, os.Getenv(tt.expect.name))
		os.Clearenv()
	}
}

func (i *ai) GetENV(p []byte) error {
	if bytes.Equal(p, []byte("yes")) {
		*i = 1
		return nil
	}
	if bytes.Equal(p, []byte("no")) {
		*i = -1
		return nil
	}
	return nil
}

func Test_GetENV(t *testing.T) {
	tests := []struct {
		env    env
		input  *assert
		expect *assert
	}{
		{
			env:    env{name: "AI", value: "unknown"},
			input:  &assert{},
			expect: &assert{AI: 0},
		},
		{
			env:    env{name: "AI", value: "yes"},
			input:  &assert{},
			expect: &assert{AI: 1},
		},
		{
			env:    env{name: "AI", value: "no"},
			input:  &assert{},
			expect: &assert{AI: -1},
		},
	}

	for _, tt := range tests {
		os.Clearenv()

		equal(t, nil, os.Setenv(tt.env.name, tt.env.value))

		err := Get(tt.input)
		equal(t, nil, err)
		equal(t, tt.expect.AI, tt.input.AI)
	}
}
