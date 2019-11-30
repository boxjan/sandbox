package exec

import (
	"syscall"
	_ "unsafe"
)

const (
	RLIMIT_CPU        = 0
	RLIMIT_FSIZE      = 1
	RLIMIT_DATA       = 2
	RLIMIT_STACK      = 3
	RLIMIT_CORE       = 4
	RLIMIT_RSS        = 5
	RLIMIT_NPROC      = 6
	RLIMIT_NOFILE     = 7
	RLIMIT_MEMLOCK    = 8
	RLIMIT_AS         = 9
	RLIMIT_LOCKS      = 10
	RLIMIT_SIGPENDING = 11
	RLIMIT_MSGQUEUE   = 12
	RLIMIT_NICE       = 13
	RLIMIT_RTPRIO     = 14
	RLIMIT_RTTIME     = 15
	RLIMIT_NLIMITS    = 16
)

//go:linkname runtimeBeforeFork syscall.runtime_BeforeFork
func runtimeBeforeFork()

//go:linkname runtimeAfterFork syscall.runtime_AfterFork
func runtimeAfterFork()

//go:linkname runtimeAfterForkInChild syscall.runtime_AfterForkInChild
func runtimeAfterForkInChild()

//go:linkname rawSyscallNoError syscall.rawSyscallNoError
func rawSyscallNoError(trap, a1, a2, a3 uintptr) (r1, r2 uintptr)

//go:linkname rawVforkSyscall syscall.rawVforkSyscall
func rawVforkSyscall(trap, a1 uintptr) (r1 uintptr, err syscall.Errno)

//go:linkname Syscall syscall.Syscall
func Syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:linkname Syscall6 syscall.Syscall6
func Syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:linkname RawSyscall syscall.RawSyscall
func RawSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:linkname RawSyscall6 syscall.RawSyscall6
func RawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:linkname fcntl syscall.fcntl
func fcntl(fd int, cmd int, arg int) (val int, err error)

//go:linkname readlen syscall.readlen
func readlen(fd int, p *byte, np int) (n int, err error)

const (
	SYS_CLONE = syscall.SYS_CLONE

	SIGCHLD = syscall.SIGCHLD

	PTRACE_EVENT_SECCOMP  = 7
	PTRACE_O_TRACESECCOMP = 1 << PTRACE_EVENT_SECCOMP

	PR_SET_NO_NEW_PRIVS = 38
	PR_GET_NO_NEW_PRIVS = 39
)
