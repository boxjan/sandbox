package main

import (
	"github.com/sdibtacm/sandbox/mods/exec"
	"syscall"
)

type SandboxRunResult struct {
	// all time setting is use millisecond
	// cpu time include kernel time and user time.
	// memory use 1kb = 1024Bytes
	KernelTime uint
	UserTime   uint
	ClockTime  uint
	UsedMemory uint64
	ExitCode   int
	StatusCode syscall.WaitStatus
	Error      exec.Error
}

func getEmptySandboxRunResult() *SandboxRunResult {
	return &SandboxRunResult{
		KernelTime: 0,
		UserTime:   0,
		ClockTime:  0,
		UsedMemory: 0,
		ExitCode:   0,
		StatusCode: 0,
		Error:      exec.Error{ErrorNum: 0, Helper: ""},
	}
}
