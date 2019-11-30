package main

import (
	"strconv"
)

type FileError struct {
	// Name is the file name for which the error occurred.
	Name string
	// Err is the underlying error.
	Err error
}

func (e *FileError) Error() string {
	return "sandbox: " + strconv.Quote(e.Name) + ": " + e.Err.Error()
}
