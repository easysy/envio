# envio

![https://img.shields.io/github/v/tag/easysy/envio](https://img.shields.io/github/v/tag/easysy/envio)
![https://img.shields.io/github/license/easysy/envio](https://img.shields.io/github/license/easysy/envio)

`envio` is a library designed to get/set environment variables to/from go structures.

## Installation

`envio` can be installed like any other Go library through `go get`:

```console
go get github.com/easysy/envio@latest
```

## Getting Started

```go
package main

import (
	"fmt"
	"os"

	"github.com/easysy/envio"
)

type Type struct {
	// will be got/set by name 'A'
	A string
	// `env:"ENV_B"` - will be got/set by name 'ENV_B'
	B string `env:"ENV_B"`
	// `env:"-"` - will be skipped
	C string `env:"-"`
	// `env:"ENV_D,m"` - will be got/set by name 'ENV_D',
	// mandatory field - if the environment doesn't contain a variable with the specified name,
	// it returns an error: the required variable $name is missing
	D   string `env:"ENV_D,m"`
	Bs  []byte `env:"ENV_BYTES"`
	// `env:"ENV_BYTES_RAW,raw"` - will be got/set by name 'ENV_BYTES_RAW',
	// since in Golang byte and uint8 mean the same thing,
	// to obtain and save an array/slice of bytes, use the 'raw' key in the tag
	BsR []byte `env:"ENV_BYTES_RAW,raw"`
}

func main() {
	in := new(Type)
	in.A = "fa"
	in.B = "fb"
	in.C = "fc"
	in.D = "fd"
	in.Bs = []byte{65, 66, 67}
	in.BsR = []byte{65, 66, 67}

	err := envio.Set(in)
	if err != nil {
		panic(err)
	}

	fmt.Printf("A: %s; ENV_B: %s; ENV_D: %s; ENV_BYTES: %s; ENV_BYTES_RAW: %s\n", os.Getenv("A"), os.Getenv("ENV_B"), os.Getenv("ENV_D"), os.Getenv("ENV_BYTES"), os.Getenv("ENV_BYTES_RAW"))
	// A: fa; ENV_B: fb; ENV_D: fd; ENV_BYTES: 65:66:67; ENV_BYTES_RAW: ABC

	out := new(Type)

	err = envio.Get(out)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", out)
	// &{A:fa B:fb C: D:fd Bs:[65 66 67] BsR:[65 66 67]}
}

```

The library supports pointers to various types.

> WARNING! Keep in mind that when using []*byte, the 'raw' flag will be ignored.