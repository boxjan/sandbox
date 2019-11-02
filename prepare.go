// +build linux

package main

import (
	"github.com/boxjan/golib/logs"
	"github.com/sdibtacm/sandbox/mods/exec"
)

func (s *SandboxRunConfig) prepare() (cmd *exec.Cmd, err error) {

	c := &exec.Cmd{Logger: logs.NewLoggerWithCmdWriterWithTraceLevel()}
	c.Logger = s.Logger
	err = c.PrepareExec(s.exec)
	if err != nil {
		return
	}

	err = c.PrepareIo(s.Io)
	if err != nil {
		return
	}

	err = c.PrepareSyscall(s.Syscall)
	if err != nil {
		return
	}

	err = c.PrepareLimit(s.Resource)
	if err != nil {
		return
	}

	err = c.PrepareCredential(s.Credential)
	if err != nil {
		return
	}

	cmd = c
	return
}
