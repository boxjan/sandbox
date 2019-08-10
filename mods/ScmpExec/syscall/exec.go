// +build linux

package scmpSyscall

import (
	scmp "github.com/sdibtacm/SandBox/mods/seccomp"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

// Lock synchronizing creation of new file descriptors with fork.
//
// We want the child in a fork/exec sequence to inherit only the
// file descriptors we intend. To do that, we mark all file
// descriptors close-on-exec and then, in the child, explicitly
// unmark the ones we want the exec'ed program to keep.
// Unix doesn't make this easy: there is, in general, no way to
// allocate a new file descriptor close-on-exec. Instead you
// have to allocate the descriptor and then mark it close-on-exec.
// If a fork happens between those two events, the child's exec
// will inherit an unwanted file descriptor.
//
// This lock solves that race: the create new fd/mark close-on-exec
// operation is done holding ForkLock for reading, and the fork itself
// is done holding ForkLock for writing. At least, that's the idea.
// There are some complications.
//
// Some system calls that create new file descriptors can block
// for arbitrarily long times: open on a hung NFS server or named
// pipe, accept on a socket, and so on. We can't reasonably grab
// the lock across those operations.
//
// It is worse to inherit some file descriptors than others.
// If a non-malicious child accidentally inherits an open ordinary file,
// that's not a big deal. On the other hand, if a long-lived child
// accidentally inherits the write end of a pipe, then the reader
// of that pipe will not see EOF until that child exits, potentially
// causing the parent program to hang. This is a common problem
// in threaded C programs that use popen.
//
// Luckily, the file descriptors that are most important not to
// inherit are not the ones that can take an arbitrarily long time
// to create: pipe returns instantly, and the net package uses
// non-blocking I/O to accept on a listening socket.
// The rules for which file descriptor-creating operations use the
// ForkLock are as follows:
//
// 1) Pipe. Does not block. Use the ForkLock.
// 2) Socket. Does not block. Use the ForkLock.
// 3) Accept. If using non-blocking mode, use the ForkLock.
//             Otherwise, live with the race.
// 4) Open. Can block. Use O_CLOEXEC if available (Linux).
//             Otherwise, live with the race.
// 5) Dup. Does not block. Use the ForkLock.
//             On Linux, could use fcntl F_DUPFD_CLOEXEC
//             instead of the ForkLock, but only for dup(fd, -1).

var ForkLock sync.RWMutex

type SysProcIDMap struct {
	ContainerID int // Container ID.
	HostID      int // Host ID.
	Size        int // Size.
}

type SysProcAttr struct {
	Chroot     string              // Chroot.
	Credential *syscall.Credential // Credential.
	// Ptrace tells the child to call ptrace(PTRACE_TRACEME).
	// Call runtime.LockOSThread before starting a process with this set,
	// and don't call UnlockOSThread until done with PtraceSyscall calls.
	Ptrace       bool
	Setsid       bool           // Create session.
	Setpgid      bool           // Set process group ID to Pgid, or, if Pgid == 0, to new pid.
	Setctty      bool           // Set controlling terminal to fd Ctty (only meaningful if Setsid is set)
	Noctty       bool           // Detach fd 0 from controlling terminal
	Ctty         int            // Controlling TTY fd
	Foreground   bool           // Place child's process group in foreground. (Implies Setpgid. Uses Ctty as fd of controlling TTY)
	Pgid         int            // Child's process group ID if Setpgid.
	Pdeathsig    syscall.Signal // Signal that the process will get when its parent dies (Linux only)
	Cloneflags   uintptr        // Flags for clone calls (Linux only)
	Unshareflags uintptr        // Flags for unshare calls (Linux only)
	UidMappings  []SysProcIDMap // User ID mappings for user namespaces.
	GidMappings  []SysProcIDMap // Group ID mappings for user namespaces.
	// GidMappingsEnableSetgroups enabling setgroups syscall.
	// If false, then setgroups syscall will be disabled for the child process.
	// This parameter is no-op if GidMappings == nil. Otherwise for unprivileged
	// users this should be set to false for mappings work.
	GidMappingsEnableSetgroups bool
	AmbientCaps                []uintptr // Ambient capabilities (Linux only)

	Scmp *scmp.ScmpFilter // Seccomp filter (Linux only)
}

var (
	none  = [...]byte{'n', 'o', 'n', 'e', 0}
	slash = [...]byte{'/', 0}
)

var zeroProcAttr ProcAttr
var zeroSysProcAttr SysProcAttr

// ProcAttr holds attributes that will be applied to a new process started
// by StartProcess.
type ProcAttr struct {
	Dir   string    // Current working directory.
	Env   []string  // Environment.
	Files []uintptr // File descriptors.
	Sys   *SysProcAttr

	Scmp *scmp.ScmpFilter // Seccomp filter
}

// StartProcess wraps ForkExec for package os.
func StartProcess(argv0 string, argv []string, attr *ProcAttr) (pid int, handle uintptr, err error) {
	pid, err = forkExec(argv0, argv, attr)
	return pid, 0, err
}

func forkExec(argv0 string, argv []string, attr *ProcAttr) (pid int, err error) {
	var p [2]int
	var n int
	var err1 syscall.Errno
	var wstatus syscall.WaitStatus

	if attr == nil {
		attr = &zeroProcAttr
	}
	sys := attr.Sys
	if sys == nil {
		sys = &zeroSysProcAttr
	}

	p[0] = -1
	p[1] = -1

	// Convert args to C form.
	argv0p, err := syscall.BytePtrFromString(argv0)
	if err != nil {
		return 0, err
	}
	argvp, err := syscall.SlicePtrFromStrings(argv)
	if err != nil {
		return 0, err
	}
	envvp, err := syscall.SlicePtrFromStrings(attr.Env)
	if err != nil {
		return 0, err
	}

	if (runtime.GOOS == "freebsd" || runtime.GOOS == "dragonfly") && len(argv[0]) > len(argv0) {
		argvp[0] = argv0p
	}

	var chroot *byte
	if sys.Chroot != "" {
		chroot, err = syscall.BytePtrFromString(sys.Chroot)
		if err != nil {
			return 0, err
		}
	}
	var dir *byte
	if attr.Dir != "" {
		dir, err = syscall.BytePtrFromString(attr.Dir)
		if err != nil {
			return 0, err
		}
	}

	// Acquire the fork lock so that no other threads
	// create new fds that are not yet close-on-exec
	// before we fork.
	ForkLock.Lock()

	// Allocate child status pipe close on exec.
	if err = forkExecPipe(p[:]); err != nil {
		goto error
	}

	// Kick off child.
	pid, err1 = forkAndExecInChild(argv0p, argvp, envvp, chroot, dir, attr, sys, p[1])
	if err1 != 0 {
		err = syscall.Errno(err1)
		goto error
	}
	//ForkLock.Unlock()

	// Read child error status from pipe.
	_ = syscall.Close(p[1])
	n, err = readlen(p[0], (*byte)(unsafe.Pointer(&err1)), int(unsafe.Sizeof(err1)))
	_ = syscall.Close(p[0])
	if err != nil || n != 0 {
		if n == int(unsafe.Sizeof(err1)) {
			err = syscall.Errno(err1)
		}
		if err == nil {
			err = syscall.EPIPE
		}

		// Child failed; wait for it to exit, to make sure
		// the zombies don't accumulate.
		_, err1 := syscall.Wait4(pid, &wstatus, 0, nil)
		for err1 == syscall.EINTR {
			_, err1 = syscall.Wait4(pid, &wstatus, 0, nil)
		}
		return 0, err
	}

	// Read got EOF, so pipe closed on exec, so exec succeeded.
	return pid, nil

error:
	if p[0] >= 0 {
		_ = syscall.Close(p[0])
		_ = syscall.Close(p[1])
	}
	ForkLock.Unlock()
	return 0, err
}

// Fork, dup fd onto 0..len(fd), and exec(argv0, argvv, envv) in child.
// If a dup or exec fails, write the errno error to pipe.
// (Pipe is close-on-exec so if exec succeeds, it will be closed.)
// In the child, this function must not acquire any locks, because
// they might have been locked at the time of the fork. This means
// no rescheduling, no malloc calls, and no new stack segments.
// For the same reason compiler does not race instrument it.
// The calls to RawSyscall are okay because they are assembly
// functions that do not grow the stack.
//go:norace
func forkAndExecInChild(argv0 *byte, argv, envv []*byte, chroot, dir *byte, attr *ProcAttr, sys *SysProcAttr, pipe int) (pid int, err syscall.Errno) {
	// Set up and fork. This returns immediately in the parent or
	// if there's an error.
	r1, err1, p, locked := forkAndExecInChild1(argv0, argv, envv, chroot, dir, attr, sys, pipe)
	if locked {
		afterFork()
	}
	if err1 != 0 {
		return 0, err1
	}

	// parent; return PID
	pid = int(r1)

	if sys.UidMappings != nil || sys.GidMappings != nil {
		_ = syscall.Close(p[0])
		err := writeUidGidMappings(pid, sys)
		var err2 syscall.Errno
		if err != nil {
			err2 = err.(syscall.Errno)
		}
		_, _, _ = syscall.RawSyscall(syscall.SYS_WRITE, uintptr(p[1]), uintptr(unsafe.Pointer(&err2)), unsafe.Sizeof(err2))
		_ = syscall.Close(p[1])
	}

	return pid, 0
}

// forkAndExecInChild1 implements the body of forkAndExecInChild up to
// the parent's post-fork path. This is a separate function so we can
// separate the child's and parent's stack frames if we're using
// vfork.
//
// This is go:noinline because the point is to keep the stack frames
// of this and forkAndExecInChild separate.
//
//go:noinline
//go:norace
func forkAndExecInChild1(argv0 *byte, argv, envv []*byte, chroot, dir *byte, attr *ProcAttr, sys *SysProcAttr, pipe int) (r1 uintptr, err1 syscall.Errno, p [2]int, locked bool) {
	// Defined in linux/prctl.h starting with Linux 4.3.
	const (
		PR_CAP_AMBIENT       = 0x2f
		PR_CAP_AMBIENT_RAISE = 0x2
	)

	// vfork requires that the child not touch any of the parent's
	// active stack frames. Hence, the child does all post-fork
	// processing in this stack frame and never returns, while the
	// parent returns immediately from this frame and does all
	// post-fork processing in the outer frame.
	// Declare all variables at top in case any
	// declarations require heap allocation (e.g., err1).
	var (
		err2   syscall.Errno
		nextfd int
		i      int
	)

	// Record parent PID so child can test if it has died.
	ppid, _ := rawSyscallNoError(syscall.SYS_GETPID, 0, 0, 0)

	// Guard against side effects of shuffling fds below.
	// Make sure that nextfd is beyond any currently open files so
	// that we can't run the risk of overwriting any of them.
	fd := make([]int, len(attr.Files))
	nextfd = len(attr.Files)
	for i, ufd := range attr.Files {
		if nextfd < int(ufd) {
			nextfd = int(ufd)
		}
		fd[i] = int(ufd)
	}
	nextfd++

	// Allocate another pipe for parent to child communication for
	// synchronizing writing of User ID/Group ID mappings.
	if sys.UidMappings != nil || sys.GidMappings != nil {
		if err := forkExecPipe(p[:]); err != nil {
			err1 = err.(syscall.Errno)
			return
		}
	}

	// About to call fork.
	// No more allocation or calls of non-assembly functions.
	beforeFork()
	locked = true
	switch {
	case runtime.GOARCH == "amd64" && sys.Cloneflags&syscall.CLONE_NEWUSER == 0:
		r1, err1 = rawVforkSyscall(syscall.SYS_CLONE, uintptr(syscall.SIGCHLD|syscall.CLONE_VFORK|syscall.CLONE_VM)|sys.Cloneflags)
	case runtime.GOARCH == "s390x":
		r1, _, err1 = syscall.RawSyscall6(syscall.SYS_CLONE, 0, uintptr(syscall.SIGCHLD)|sys.Cloneflags, 0, 0, 0, 0)
	default:
		r1, _, err1 = syscall.RawSyscall6(syscall.SYS_CLONE, uintptr(syscall.SIGCHLD)|sys.Cloneflags, 0, 0, 0, 0, 0)
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

	afterForkInChild()

	// Enable the "keep capabilities" flag to set ambient capabilities later.
	if len(sys.AmbientCaps) > 0 {
		_, _, err1 = syscall.RawSyscall6(syscall.SYS_PRCTL, syscall.PR_SET_KEEPCAPS, 1, 0, 0, 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// Wait for User ID/Group ID mappings to be written.
	if sys.UidMappings != nil || sys.GidMappings != nil {
		if _, _, err1 = syscall.RawSyscall(syscall.SYS_CLOSE, uintptr(p[1]), 0, 0); err1 != 0 {
			goto childerror
		}
		r1, _, err1 = syscall.RawSyscall(syscall.SYS_READ, uintptr(p[0]), uintptr(unsafe.Pointer(&err2)), unsafe.Sizeof(err2))
		if err1 != 0 {
			goto childerror
		}
		if r1 != unsafe.Sizeof(err2) {
			err1 = syscall.EINVAL
			goto childerror
		}
		if err2 != 0 {
			err1 = err2
			goto childerror
		}
	}

	// Session ID
	if sys.Setsid {
		_, _, err1 = syscall.RawSyscall(syscall.SYS_SETSID, 0, 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// Set process group
	if sys.Setpgid || sys.Foreground {
		// Place child in process group.
		_, _, err1 = syscall.RawSyscall(syscall.SYS_SETPGID, 0, uintptr(sys.Pgid), 0)
		if err1 != 0 {
			goto childerror
		}
	}

	if sys.Foreground {
		pgrp := int32(sys.Pgid)
		if pgrp == 0 {
			r1, _ = rawSyscallNoError(syscall.SYS_GETPID, 0, 0, 0)

			pgrp = int32(r1)
		}

		// Place process group in foreground.
		_, _, err1 = syscall.RawSyscall(syscall.SYS_IOCTL, uintptr(sys.Ctty), uintptr(syscall.TIOCSPGRP), uintptr(unsafe.Pointer(&pgrp)))
		if err1 != 0 {
			goto childerror
		}
	}

	// Unshare
	if sys.Unshareflags != 0 {
		_, _, err1 = syscall.RawSyscall(syscall.SYS_UNSHARE, sys.Unshareflags, 0, 0)
		if err1 != 0 {
			goto childerror
		}
		// The unshare system call in Linux doesn't unshare mount points
		// mounted with --shared. Systemd mounts / with --shared. For a
		// long discussion of the pros and cons of this see debian bug 739593.
		// The Go model of unsharing is more like Plan 9, where you ask
		// to unshare and the namespaces are unconditionally unshared.
		// To make this model work we must further mark / as MS_PRIVATE.
		// This is what the standard unshare command does.
		if sys.Unshareflags&syscall.CLONE_NEWNS == syscall.CLONE_NEWNS {
			_, _, err1 = syscall.RawSyscall6(syscall.SYS_MOUNT, uintptr(unsafe.Pointer(&none[0])), uintptr(unsafe.Pointer(&slash[0])), 0, syscall.MS_REC|syscall.MS_PRIVATE, 0, 0)
			if err1 != 0 {
				goto childerror
			}
		}
	}

	// Chroot
	if chroot != nil {
		_, _, err1 = syscall.RawSyscall(syscall.SYS_CHROOT, uintptr(unsafe.Pointer(chroot)), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// User and groups
	if cred := sys.Credential; cred != nil {
		ngroups := uintptr(len(cred.Groups))
		groups := uintptr(0)
		if ngroups > 0 {
			groups = uintptr(unsafe.Pointer(&cred.Groups[0]))
		}
		if !(sys.GidMappings != nil && !sys.GidMappingsEnableSetgroups && ngroups == 0) && !cred.NoSetGroups {
			_, _, err1 = syscall.RawSyscall(syscall.SYS_SETGROUPS, ngroups, groups, 0)
			if err1 != 0 {
				goto childerror
			}
		}
		_, _, err1 = syscall.RawSyscall(syscall.SYS_SETGID, uintptr(cred.Gid), 0, 0)
		if err1 != 0 {
			goto childerror
		}
		_, _, err1 = syscall.RawSyscall(syscall.SYS_SETUID, uintptr(cred.Uid), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	for _, c := range sys.AmbientCaps {
		_, _, err1 = syscall.RawSyscall6(syscall.SYS_PRCTL, PR_CAP_AMBIENT, uintptr(PR_CAP_AMBIENT_RAISE), c, 0, 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// Chdir
	if dir != nil {
		_, _, err1 = syscall.RawSyscall(syscall.SYS_CHDIR, uintptr(unsafe.Pointer(dir)), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// Parent death signal
	if sys.Pdeathsig != 0 {
		_, _, err1 = syscall.RawSyscall6(syscall.SYS_PRCTL, syscall.PR_SET_PDEATHSIG, uintptr(sys.Pdeathsig), 0, 0, 0, 0)
		if err1 != 0 {
			goto childerror
		}

		// Signal self if parent is already dead. This might cause a
		// duplicate signal in rare cases, but it won't matter when
		// using SIGKILL.
		r1, _ = rawSyscallNoError(syscall.SYS_GETPPID, 0, 0, 0)
		if r1 != ppid {
			pid, _ := rawSyscallNoError(syscall.SYS_GETPID, 0, 0, 0)
			_, _, err1 := syscall.RawSyscall(syscall.SYS_KILL, pid, uintptr(sys.Pdeathsig), 0)
			if err1 != 0 {
				goto childerror
			}
		}
	}

	// Pass 1: look for fd[i] < i and move those up above len(fd)
	// so that pass 2 won't stomp on an fd it needs later.
	if pipe < nextfd {
		_, _, err1 = syscall.RawSyscall(syscall.SYS_DUP, uintptr(pipe), uintptr(nextfd), 0)
		if err1 != 0 {
			goto childerror
		}
		_, _, _ = syscall.RawSyscall(syscall.SYS_FCNTL, uintptr(nextfd), syscall.F_SETFD, syscall.FD_CLOEXEC)
		pipe = nextfd
		nextfd++
	}
	for i = 0; i < len(fd); i++ {
		if fd[i] >= 0 && fd[i] < int(i) {
			if nextfd == pipe { // don't stomp on pipe
				nextfd++
			}
			_, _, err1 = syscall.RawSyscall(syscall.SYS_DUP, uintptr(fd[i]), uintptr(nextfd), 0)
			if err1 != 0 {
				goto childerror
			}
			_, _, _ = syscall.RawSyscall(syscall.SYS_FCNTL, uintptr(nextfd), syscall.F_SETFD, syscall.FD_CLOEXEC)
			fd[i] = nextfd
			nextfd++
		}
	}

	// Pass 2: dup fd[i] down onto i.
	for i = 0; i < len(fd); i++ {
		if fd[i] == -1 {
			_, _, _ = syscall.RawSyscall(syscall.SYS_CLOSE, uintptr(i), 0, 0)
			continue
		}
		if fd[i] == int(i) {
			// dup2(i, i) won't clear close-on-exec flag on Linux,
			// probably not elsewhere either.
			_, _, err1 = syscall.RawSyscall(syscall.SYS_FCNTL, uintptr(fd[i]), syscall.F_SETFD, 0)
			if err1 != 0 {
				goto childerror
			}
			continue
		}
		// The new fd is created NOT close-on-exec,
		// which is exactly what we want.
		_, _, err1 = syscall.RawSyscall(syscall.SYS_DUP, uintptr(fd[i]), uintptr(i), 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// By convention, we don't close-on-exec the fds we are
	// started with, so if len(fd) < 3, close 0, 1, 2 as needed.
	// Programs that know they inherit fds >= 3 will need
	// to set them close-on-exec.
	for i = len(fd); i < 3; i++ {
		_, _, _ = syscall.RawSyscall(syscall.SYS_CLOSE, uintptr(i), 0, 0)
	}

	// Detach fd 0 from tty
	if sys.Noctty {
		_, _, err1 = syscall.RawSyscall(syscall.SYS_IOCTL, 0, uintptr(syscall.TIOCNOTTY), 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// Set the controlling TTY to Ctty
	if sys.Setctty {
		_, _, err1 = syscall.RawSyscall(syscall.SYS_IOCTL, uintptr(sys.Ctty), uintptr(syscall.TIOCSCTTY), 1)
		if err1 != 0 {
			goto childerror
		}
	}

	// Enable tracing if requested.
	// Do this right before exec so that we don't unnecessarily trace the runtime
	// setting up after the fork. See issue #21428.
	if sys.Ptrace {
		_, _, err1 = syscall.RawSyscall(syscall.SYS_PTRACE, uintptr(syscall.PTRACE_TRACEME), 0, 0)
		if err1 != 0 {
			goto childerror
		}
	}

	// Load Seccomp filter before exec
	if sys.Scmp != nil {
		err1 = sys.Scmp.LoadWithoutCheck()
		//_ = sys.Scmp.Load()

		if err1 != 0 {
			goto childerror
		}
		//sys.Scmp.Release()
	}

	// Time to exec.
	_, _, err1 = syscall.RawSyscall(syscall.SYS_EXECVE,
		uintptr(unsafe.Pointer(argv0)),
		uintptr(unsafe.Pointer(&argv[0])),
		uintptr(unsafe.Pointer(&envv[0])))

childerror:
	// send error code on pipe
	syscall.RawSyscall(syscall.SYS_WRITE, uintptr(pipe), uintptr(unsafe.Pointer(&err1)), unsafe.Sizeof(err1))
	for {
		_, _, _ = syscall.RawSyscall(syscall.SYS_EXIT, 253, 0, 0)
	}
}

// Try to open a pipe with O_CLOEXEC set on both file descriptors.
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
