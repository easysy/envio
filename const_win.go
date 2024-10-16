//go:build windows

package envio

const envSeparator = ';'

const (
	testEnvSlc   = "true;false;true"
	testEnvArr   = "0;5;8;0;0"
	testEnvBytes = "65;66;67;68"
)
