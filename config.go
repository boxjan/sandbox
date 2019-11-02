package main

import (
	"github.com/boxjan/golib/logs"
	"github.com/sdibtacm/sandbox/mods/exec"
	"sync"
)

type SandboxRunConfig struct {
	sync.Mutex
	running bool

	Logger     *logs.Logger
	exec       *exec.ExecSetting
	Io         *exec.IoSetting
	Resource   *exec.ResourceSetting
	Credential *exec.Credential
	Syscall    *exec.SyscallSetting
}

func GetDefaultSandboxRunConfig() *SandboxRunConfig {
	return &SandboxRunConfig{
		running:    false,
		Logger:     logs.NewLoggerWithCmdWriter(logs.LevelDebugStr),
		exec:       exec.GetDefaultExecSetting(),
		Io:         exec.GetDefaultIoSetting(),
		Resource:   exec.GetDefaultResourceSetting(),
		Credential: exec.GetDefaultCredential(),
		Syscall:    exec.GetDefaultSyscallSetting(),
	}
}

func GetEmptySandboxRunConfig() *SandboxRunConfig {
	return &SandboxRunConfig{
		running: false,
		Logger:  logs.NewLoggerWithCmdWriter(logs.LevelDebugStr),
		exec: &exec.ExecSetting{
			Path: "",
			Args: nil,
			Env:  nil,
		},
		Io:         exec.GetDefaultIoSetting(),
		Resource:   exec.GetUnlimitResourceSetting(),
		Credential: exec.GetDefaultCredential(),
		Syscall: &exec.SyscallSetting{
			RunLevel:               0,
			UsePtraceGetBadSyscall: true,
		},
	}
}

func GetDefaultResourceSetting() *exec.ResourceSetting {
	return exec.GetDefaultResourceSetting()
}

func GetUnlimitResourceSetting() *exec.ResourceSetting {
	return exec.GetUnlimitResourceSetting()
}

func GetDefaultIoSetting() *exec.IoSetting {
	return exec.GetDefaultIoSetting()
}

func GetDefaultExecSetting() *exec.ExecSetting {
	return exec.GetDefaultExecSetting()
}

func GetDefaultSyscallSetting() *exec.SyscallSetting {
	return exec.GetDefaultSyscallSetting()
}
