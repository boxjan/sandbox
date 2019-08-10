package main

import (
	"syscall"
	"time"
)

type RuntimeLimit struct {
	clockTime   uint
	cpuTime     uint
	memory      int64
	threadLimit int64
}

func (config *RunConfig) memoryLimit() {
	config.log.Debug("memory killer up")
	for {
		if syscall.Kill(config.pid, 0) != nil {
			break
		}
		time.Sleep(5 * time.Microsecond)
	}
}
