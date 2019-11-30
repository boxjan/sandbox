package exec

import (
	_ "unsafe"
)

//go:linkname itoa os.itoa
func itoa(val int) string
