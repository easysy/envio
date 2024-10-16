package envio

import "reflect"

var (
	getter = reflect.TypeOf((*Getter)(nil)).Elem()
	setter = reflect.TypeOf((*Setter)(nil)).Elem()
)

// Getter is the interface implemented by types that can themselves get ENVs.
type Getter interface {
	GetENV([]byte) error
}

// Setter is the interface implemented by types that can themselves set ENVs.
type Setter interface {
	SetENV() ([]byte, error)
}
