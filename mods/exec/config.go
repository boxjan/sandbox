package exec

import (
	"io"
	"os"
	"syscall"
)

type ExecAttr struct {
	Path  *byte
	Args  []*byte
	Env   []*byte
	Chdir *byte

	Files []uintptr

	Uid int
	Gid int

	OutputSizeRlimit *syscall.Rlimit
	Cloneflags       uintptr
	ScmpBpf          *syscall.SockFprog

	Pdeathsig syscall.Signal // Signal that the process will get when its parent dies (Linux only)

	SetUid        bool
	SetGid        bool
	SetNoNewPrivs bool
	Ptrace        bool
}

const RESOURCE_UNLIMIT = 0xFFFFFFFFFFFFFFFF

type ResourceSetting struct {
	// all time setting is use millisecond
	// cpu time include kernel time and user time.
	// memory setting use 1kb = 1024Bytes
	ClockTime  uint
	CpuTime    uint64 // will include kernel time and user time
	Memory     uint64 // I think 4T memory is enough
	OutputSize uint64 // byte, will use rlimit->FSIZE
	Thread     uint64 // the sum of threads for all processes
}

func GetDefaultResourceSetting() *ResourceSetting {
	return &ResourceSetting{
		ClockTime:  1000,
		CpuTime:    1000,
		Memory:     16 * 1024, // 16 MB
		OutputSize: 1 * 1024,  // 1K
		Thread:     8,
	}
}

func GetUnlimitResourceSetting() *ResourceSetting {
	return &ResourceSetting{
		ClockTime:  0x7FFFFFFF,
		CpuTime:    RESOURCE_UNLIMIT,
		Memory:     RESOURCE_UNLIMIT,
		OutputSize: RESOURCE_UNLIMIT,
		Thread:     RESOURCE_UNLIMIT,
	}
}

type IoSetting struct {
	// input only have stdin
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func GetDefaultIoSetting() *IoSetting {

	return &IoSetting{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Credential holds user and group identities to be assumed
type Credential struct {
	Uid int // User ID.
	Gid int // Group ID.
}

func GetDefaultCredential() *Credential {

	if syscall.Getuid() == 0 {
		return &Credential{
			Uid: 65534, // nobody
			Gid: 65534,
		}
	} else {
		return nil
	}
}

type ExecSetting struct {
	Path string
	Args []string
	Env  []string
	Dir  string
}

func GetDefaultExecSetting() *ExecSetting {
	return &ExecSetting{}
}

type RunPermissionLevel int8

const (
	// see the docs/running-level.md know more about it
	SandboxRunLevel0 RunPermissionLevel = iota
	SandboxRunLevel1
	SandboxRunLevel2
	SandboxRunLevel3
	SandboxRunLevel4
	SandboxRunLevel5
	SandboxRunLevel6
	SandboxRunLevel7
)

type SyscallSetting struct {
	RunLevel               RunPermissionLevel
	UsePtraceGetBadSyscall bool
}

func GetDefaultSyscallSetting() *SyscallSetting {
	return &SyscallSetting{
		RunLevel:               SandboxRunLevel0,
		UsePtraceGetBadSyscall: false,
	}
}
