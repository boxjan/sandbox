// +build linux

package exec

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var ForkLock sync.RWMutex

//go:noinline
//go:norace
func forkAndExecInChild(attr *ExecAttr, pipe int) (r1 uintptr, err1 syscall.Errno, p [2]int, locked bool) {

	var (
		nextfd int
		i      int
	)

	pathPtr := attr.Path
	argsPtr := &attr.Args[0]
	envPtr := &attr.Env[0]
	// Record parent PID so child can test if it has died.
	ppid, _ := rawSyscallNoError(syscall.SYS_GETPID, 0, 0, 0)

	fd := make([]int, len(attr.Files))
	nextfd = len(attr.Files)
	for i, ufd := range attr.Files {
		if nextfd < int(ufd) {
			nextfd = int(ufd)
		}
		fd[i] = int(ufd)
	}
	nextfd++

	hasRawVforkSyscall := runtime.GOARCH == "amd64" || runtime.GOARCH == "ppc64" || runtime.GOARCH == "s390x" || runtime.GOARCH == "arm64"
	// About to call fork.
	// No more allocation or calls of non-assembly functions.
	runtimeBeforeFork()
	locked = true
	switch {
	case hasRawVforkSyscall && (attr.Cloneflags&syscall.CLONE_NEWUSER == 0):
		r1, err1 = rawVforkSyscall(SYS_CLONE, uintptr(SIGCHLD|CLONE_VFORK|CLONE_VM)|attr.Cloneflags)
	case runtime.GOARCH == "s390x":
		r1, _, err1 = syscall.RawSyscall6(SYS_CLONE, 0, uintptr(SIGCHLD)|attr.Cloneflags, 0, 0, 0, 0)
	default:
		r1, _, err1 = syscall.RawSyscall6(SYS_CLONE, uintptr(SIGCHLD)|attr.Cloneflags, 0, 0, 0, 0, 0)
	}
	if err1 != 0 || r1 != 0 {
		// If we're in the parent, we must return immediately
		// so we're not in the same stack frame as the child.
		// This can at most use the return PC, which the child
		// will not modify, and the results of
		// rawVforkSyscall, which must have been written after
		// the child was replaced.
		return
	}

	runtimeAfterForkInChild()

	if attr.Chdir != nil {
		_, _, err1 = RawSyscall(syscall.SYS_CHDIR, uintptr(unsafe.Pointer(attr.Chdir)), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// Parent death signal
	if attr.Pdeathsig != 0 {
		_, _, err1 = syscall.RawSyscall6(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(attr.Pdeathsig), 0, 0, 0, 0)
		if err1 != 0 {
			goto childerror
		}

		// Signal self if parent is already dead. This might cause a
		// duplicate signal in rare cases, but it won't matter when
		// using SIGKILL.
		r1, _ = rawSyscallNoError(syscall.SYS_GETPPID, 0, 0, 0)
		if r1 != ppid {
			pid, _ := rawSyscallNoError(syscall.SYS_GETPID, 0, 0, 0)
			_, _, err1 := syscall.RawSyscall(syscall.SYS_KILL, pid, uintptr(attr.Pdeathsig), 0)
			if err1 != 0 {
				goto childerror
			}
		}
	}

	if pipe < nextfd {
		_, _, err1 = RawSyscall(syscall.SYS_DUP, uintptr(pipe), uintptr(nextfd), 0)
		if err1 != 0 {
			goto childerror
		}
		_, _, _ = RawSyscall(syscall.SYS_FCNTL, uintptr(nextfd), syscall.F_SETFD, syscall.FD_CLOEXEC)
		pipe = nextfd
		nextfd++
	}
	for i = 0; i < len(fd); i++ {
		if fd[i] >= 0 && fd[i] < int(i) {
			if nextfd == pipe { // don't stomp on pipe
				nextfd++
			}
			_, _, err1 = RawSyscall(syscall.SYS_DUP, uintptr(fd[i]), uintptr(nextfd), 0)
			if err1 != 0 {
				goto childerror
			}
			_, _, _ = RawSyscall(syscall.SYS_FCNTL, uintptr(nextfd), syscall.F_SETFD, syscall.FD_CLOEXEC)
			fd[i] = nextfd
			nextfd++
		}
	}

	// Pass 2: dup fd[i] down onto i.
	for i = 0; i < len(fd); i++ {
		if fd[i] == -1 {
			_, _, _ = RawSyscall(syscall.SYS_CLOSE, uintptr(i), 0, 0)
			continue
		}
		if fd[i] == int(i) {
			// dup2(i, i) won't clear close-on-exec flag on Linux,
			// probably not elsewhere either.
			_, _, err1 = RawSyscall(syscall.SYS_FCNTL, uintptr(fd[i]), syscall.F_SETFD, 0)
			if err1 != 0 {
				goto childerror
			}
			continue
		}
		// The new fd is created NOT close-on-exec,
		// which is exactly what we want.
		_, _, err1 = RawSyscall(syscall.SYS_DUP, uintptr(fd[i]), uintptr(i), 0)
		if err1 != 0 {
			goto childerror
		}
	}

	if attr.OutputSizeRlimit != nil {
		_, _, err1 = RawSyscall(syscall.SYS_SETRLIMIT, uintptr(syscall.RLIMIT_FSIZE), uintptr(unsafe.Pointer(attr.OutputSizeRlimit)), 0)
		if err1 != 0 {
			goto childerror
		}
	}

	//if attr.Ptrace {
	//	_, _, err1 = RawSyscall(syscall.SYS_PTRACE, uintptr(syscall.PTRACE_TRACEME), 0, 0)
	//	if err1 != 0 {
	//		goto childerror
	//	}
	//}

	if attr.SetNoNewPrivs {
		_, _, err1 = RawSyscall(syscall.SYS_PRCTL, PR_SET_NO_NEW_PRIVS, 1, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	if attr.SetGid {
		_, _, err1 = RawSyscall(syscall.SYS_SETGID, uintptr(attr.Gid), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	if attr.SetUid {
		_, _, err1 = RawSyscall(syscall.SYS_SETUID, uintptr(attr.Uid), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	if attr.ScmpBpf != nil {
		_, _, err1 = RawSyscall6(syscall.SYS_PRCTL, syscall.PR_SET_SECCOMP, 2, uintptr(unsafe.Pointer(attr.ScmpBpf)), 0, 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	_, _, err1 = RawSyscall(syscall.SYS_EXECVE,
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		uintptr(unsafe.Pointer(envPtr)))

childerror:
	// send error code on pipe
	RawSyscall(syscall.SYS_WRITE, uintptr(pipe), uintptr(unsafe.Pointer(&err1)), unsafe.Sizeof(err1))
	for {
		_, _, _ = RawSyscall(syscall.SYS_EXIT, 253, 0, 0)
	}
}
