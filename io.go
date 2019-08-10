package main

import (
	"io"
)

type RuntimeIO struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}
