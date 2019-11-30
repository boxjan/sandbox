//+build linux

package exec

import (
	"errors"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var ForkLock sync.RWMutex

type SysAttr struct {
	Ptrace        bool
	Setsid        bool
	RlimitList    [20]uint64
	SetNoNewPrivs bool
	Cloneflags    uintptr
	Files         []uintptr
	Pdeathsig     uint
	Credential    *Credential
	Bpf           *syscall.SockFprog
}

type Credential struct {
	Uid   int
	Gid   int
	Umask uint
}

type ExecError struct {
	Step int
	Err  error
}

func (e *ExecError) Error() string {
	return "exec: step[" + SANDBOX_STEP_STR[e.Step] + "] with error: [" + e.Err.Error() + "]"
}

var zeroSysAttr SysAttr

func forkExec(argv0 *byte, argv, envv []*byte, chroot, dir *byte, attr *SysAttr) (pid int, err error) {

	var (
		stepPipe [2]int
		errPipe  [2]int
		stepN    int
		errN     int
		err1     syscall.Errno
		err2     error
		err3     error
		wstatus  syscall.WaitStatus
		step     int
	)

	ForkLock.Lock()
	if err = forkExecPipe(errPipe[:]); err != nil {
		goto error
	}
	if err = forkExecPipe(stepPipe[:]); err != nil {
		goto error
	}

	pid, err1 = cloneAndExecInChild(argv0, argv, envv, chroot, dir, attr, errPipe[1], stepPipe[1])
	if err1 != 0 {
		err = &ExecError{Step: SANDBOX_READY_FOR_CLONE, Err: errors.New(err1.Error())}
		goto error
	}
	ForkLock.Unlock()

	// Read child error status from pipe.
	_ = syscall.Close(errPipe[1])
	errN, err2 = readlen(errPipe[0], (*byte)(unsafe.Pointer(&err1)), int(unsafe.Sizeof(err1)))
	_ = syscall.Close(errPipe[0])
	_ = syscall.Close(stepPipe[1])
	stepN, err3 = readlen(stepPipe[0], (*byte)(unsafe.Pointer(&step)), int(unsafe.Sizeof(step)))
	_ = syscall.Close(stepPipe[0])
	if err2 != nil || err3 != nil || errN != 0 {
		if errN == int(unsafe.Sizeof(err1)) && stepN == int(unsafe.Sizeof(step)) {
			err = &ExecError{Step: step, Err: errors.New(err1.Error())}
		}
		if err == nil && err2 == nil {
			err = &ExecError{Step: SANDBOX_READ_PIPE, Err: syscall.EPIPE}
		}
		if err == nil && err3 == nil {
			err = &ExecError{Step: SANDBOX_READ_PIPE, Err: syscall.EPIPE}
		}

		// Child failed; wait for it to exit, to make sure
		// the zombies don't accumulate.
		_, err1 := syscall.Wait4(pid, &wstatus, 0, nil)
		for err1 == syscall.EINTR {
			_, err1 = syscall.Wait4(pid, &wstatus, 0, nil)
		}
		return 0, err
	}
	return

error:
	if stepPipe[0] >= 0 {
		_ = syscall.Close(stepPipe[0])
		_ = syscall.Close(stepPipe[1])
	}
	if errPipe[0] >= 0 {
		_ = syscall.Close(errPipe[0])
		_ = syscall.Close(errPipe[1])
	}
	ForkLock.Unlock()
	return 0, &ExecError{Step: SANDBOX_PREPARE_PIPE, Err: err2}
}

func cloneAndExecInChild(argv0 *byte, argv, envv []*byte, chroot, dir *byte, attr *SysAttr, errPipe, stepPipe int) (pid int, err syscall.Errno) {

	r1, err1, locked := cloneAndExecInChild1(argv0, argv, envv, chroot, dir, attr, errPipe, stepPipe)
	if locked {
		runtimeAfterFork()
	}
	if err1 != 0 {
		return 0, err1
	}

	// parent; return PID
	pid = int(r1)
	return pid, 0

}

var step int = SANDBOX_NO_START

//go:noinline
//go:norace
func cloneAndExecInChild1(argv0 *byte, argv, envv []*byte, chroot, dir *byte, sys *SysAttr, errPipe, stepPipe int) (r1 uintptr, err1 syscall.Errno, locked bool) {
	// The function will do clone, load limit, exec function
	// because will no use normal function after clone,
	// to let the parent know which step is happen error,
	// will use pipe to sent step num and errno.

	var (
		//err2                      syscall.Errno
		nextfd int
		i      int
		//fd1                       uintptr
	)

	ppid, _ := rawSyscallNoError(syscall.SYS_GETPID, 0, 0, 0)

	// Guard against side effects of shuffling fds below.
	// Make sure that nextfd is beyond any currently open files so
	// that we can't run the risk of overwriting any of them.
	fd := make([]int, len(sys.Files))
	nextfd = len(sys.Files)
	for i, ufd := range sys.Files {
		if nextfd < int(ufd) {
			nextfd = int(ufd)
		}
		fd[i] = int(ufd)
	}
	nextfd++

	runtimeBeforeFork()
	locked = true

	step = SANDBOX_READY_FOR_CLONE
	switch {
	case runtime.GOARCH == "s390x":
		r1, _, err1 = RawSyscall6(SYS_CLONE, 0, uintptr(SIGCHLD)|sys.Cloneflags, 0, 0, 0, 0)
	default:
		r1, _, err1 = RawSyscall6(SYS_CLONE, uintptr(SIGCHLD)|sys.Cloneflags, 0, 0, 0, 0, 0)
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

	// Fork succeeded, now in child.

	runtimeAfterForkInChild()

	// Session ID
	if sys.Setsid {
		_, _, err1 = RawSyscall(syscall.SYS_SETSID, 0, 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// Chroot
	if chroot != nil {
		step = SANDBOX_READY_FOR_CHROOT
		_, _, err1 = RawSyscall(syscall.SYS_CHROOT, uintptr(unsafe.Pointer(chroot)), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	if cred := sys.Credential; cred != nil {
		if cred.Uid != 0 {
			step = SANDBOX_READY_FOR_SETUID
			_, _, err1 = RawSyscall(syscall.SYS_SETGID, uintptr(cred.Gid), 0, 0)
			if err1 != 0 {
				goto childerror
			}
		}
		if cred.Gid != 0 {
			step = SANDBOX_READY_FOR_SETGID
			_, _, err1 = RawSyscall(syscall.SYS_SETUID, uintptr(cred.Uid), 0, 0)
			if err1 != 0 {
				goto childerror
			}
		}
		if cred.Umask != 0 {
			step = SANDBOX_READY_FOR_SETUMASK
			_, _, err1 = RawSyscall(syscall.SYS_UMASK, uintptr(cred.Umask), 0, 0)
			if err1 != 0 {
				goto childerror
			}
		}
	}

	// Chdir
	if dir != nil {
		step = SANDBOX_READY_FOR_CHDIR
		_, _, err1 = RawSyscall(syscall.SYS_CHDIR, uintptr(unsafe.Pointer(dir)), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// Parent death signal
	if sys.Pdeathsig != 0 {
		step = SANDBOX_READY_FOR_SET_PDEATHSIG
		_, _, err1 = RawSyscall6(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(sys.Pdeathsig), 0, 0, 0, 0)
		if err1 != 0 {
			goto childerror
		}

		// Signal self if parent is already dead. This might cause a
		// duplicate signal in rare cases, but it won't matter when
		// using SIGKILL.
		r1, _ = rawSyscallNoError(syscall.SYS_GETPPID, 0, 0, 0)
		if r1 != ppid {
			pid, _ := rawSyscallNoError(syscall.SYS_GETPID, 0, 0, 0)
			step = SANDBOX_READY_FOR_PDEATHSIG_KILL_MYSELF
			_, _, err1 := RawSyscall(syscall.SYS_KILL, pid, uintptr(sys.Pdeathsig), 0)
			if err1 != 0 {
				goto childerror
			}
		}
	}

	step = SANDBOX_READY_FRO_DUP_FILE
	// Pass 1: look for fd[i] < i and move those up above len(fd)
	// so that pass 2 won't stomp on an fd it needs later.
	if errPipe < nextfd {
		_, _, err1 = RawSyscall(syscall.SYS_DUP2, uintptr(errPipe), uintptr(nextfd), 0)
		if err1 != 0 {
			goto childerror
		}
		RawSyscall(syscall.SYS_FCNTL, uintptr(nextfd), syscall.F_SETFD, syscall.FD_CLOEXEC)
		errPipe = nextfd
		nextfd++
	}
	for i = 0; i < len(fd); i++ {
		if fd[i] >= 0 && fd[i] < int(i) {
			if nextfd == errPipe { // don't stomp on pipe
				nextfd++
			}
			_, _, err1 = RawSyscall(syscall.SYS_DUP2, uintptr(fd[i]), uintptr(nextfd), 0)
			if err1 != 0 {
				goto childerror
			}
			RawSyscall(syscall.SYS_FCNTL, uintptr(nextfd), syscall.F_SETFD, syscall.FD_CLOEXEC)
			fd[i] = nextfd
			nextfd++
		}
	}

	// Pass 2: dup fd[i] down onto i.
	for i = 0; i < len(fd); i++ {
		if fd[i] == -1 {
			RawSyscall(syscall.SYS_CLOSE, uintptr(i), 0, 0)
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
		_, _, err1 = RawSyscall(syscall.SYS_DUP2, uintptr(fd[i]), uintptr(i), 0)
		if err1 != 0 {
			goto childerror
		}
	}

	step = SANDBOX_READY_FOR_SET_RLIMIT
	for i = 0; i <= RLIMIT_NLIMITS; i++ {
		if sys.RlimitList[i] != RLIMIT_UNRESOURCE {
			_, _, err1 := RawSyscall(syscall.SYS_SETRLIMIT, uintptr(i),
				uintptr(unsafe.Pointer(&syscall.Rlimit{Cur: sys.RlimitList[i], Max: sys.RlimitList[i]})), 0)
			if err1 != 0 {
				goto childerror
			}
		}
	}

	if sys.Ptrace {
		step = SANDBOX_READY_FOR_SET_PTRACE
		_, _, err1 = RawSyscall(syscall.SYS_PTRACE, uintptr(syscall.PTRACE_TRACEME), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	if sys.Bpf != nil {
		step = SANDBOX_READY_FOR_SET_BPF
		_, _, err1 = RawSyscall(syscall.SYS_PRCTL, syscall.PR_SET_SECCOMP, 2, uintptr(unsafe.Pointer(sys.Bpf)))
		if err1 != 0 {
			goto childerror
		}
	}

	// Time to exec.
	step = SANDBOX_READY_FOR_EXEC
	_, _, err1 = RawSyscall(syscall.SYS_EXECVE,
		uintptr(unsafe.Pointer(argv0)),
		uintptr(unsafe.Pointer(&argv[0])),
		uintptr(unsafe.Pointer(&envv[0])))

childerror:
	RawSyscall(syscall.SYS_WRITE, uintptr(errPipe), uintptr(unsafe.Pointer(&err1)), unsafe.Sizeof(err1))  // what error
	RawSyscall(syscall.SYS_WRITE, uintptr(stepPipe), uintptr(unsafe.Pointer(&step)), unsafe.Sizeof(step)) // which step
	for {
		_, _, _ = RawSyscall(syscall.SYS_EXIT, 253, 0, 0)
	}
}

func forkExecPipe(p []int) (err error) {
	err = syscall.Pipe2(p, syscall.O_CLOEXEC)
	// pipe2 was added in 2.6.27 and our minimum requirement is 2.6.23, so it
	// might not be implemented.
	if err == syscall.ENOSYS {
		if err = syscall.Pipe(p); err != nil {
			return
		}
		if _, err = fcntl(p[0], syscall.F_SETFD, syscall.FD_CLOEXEC); err != nil {
			return
		}
		_, err = fcntl(p[1], syscall.F_SETFD, syscall.FD_CLOEXEC)
	}
	return
}
