// +build linux

package scmpOs

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"errors"
	scmpSyscall "github.com/sdibtacm/SandBox/mods/ScmpExec/syscall"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

// ProcAttr holds the attributes that will be applied to a new process
// started by StartProcess.
type ProcAttr struct {
	// If Dir is non-empty, the child changes into the directory before
	// creating the process.
	Dir string
	// If Env is non-nil, it gives the environment variables for the
	// new process in the form returned by Environ.
	// If it is nil, the result of Environ will be used.
	Env []string
	// Files specifies the open files inherited by the new process. The
	// first three entries correspond to standard input, standard output, and
	// standard error. An implementation may support additional entries,
	// depending on the underlying operating system. A nil entry corresponds
	// to that file being closed when the process starts.
	Files []*os.File

	// Operating system-specific process creation attributes.
	// Note that setting this field means that your program
	// may not execute properly or even compile on some
	// operating systems.
	Sys *scmpSyscall.SysProcAttr
}

// The only signal values guaranteed to be present in the os package on all
// systems are os.Interrupt (send the process an interrupt) and os.Kill (force
// the process to exit). On Windows, sending os.Interrupt to a process with
// os.Process.Signal is not implemented; it will return an error instead of
// sending a signal.
var (
	Interrupt Signal = syscall.SIGINT
	Kill      Signal = syscall.SIGKILL
)

func startProcess(name string, argv []string, attr *ProcAttr) (p *Process, err error) {
	// If there is no SysProcAttr (ie. no Chroot or changed
	// UID/GID), double-check existence of the directory we want
	// to chdir into. We can make the error clearer this way.
	if attr != nil && attr.Sys == nil && attr.Dir != "" {
		if _, err := os.Stat(attr.Dir); err != nil {
			pe := err.(*os.PathError)
			pe.Op = "chdir"
			return nil, pe
		}
	}

	sysattr := &scmpSyscall.ProcAttr{
		Dir: attr.Dir,
		Env: attr.Env,
		Sys: attr.Sys,
	}
	if sysattr.Env == nil {
		sysattr.Env, err = environForSysProcAttr(sysattr.Sys)
		if err != nil {
			return nil, err
		}
	}
	for _, f := range attr.Files {
		sysattr.Files = append(sysattr.Files, f.Fd())
	}

	pid, h, e := scmpSyscall.StartProcess(name, argv, sysattr)
	if e != nil {
		return nil, &os.PathError{"fork/exec", name, e}
	}
	return newProcess(pid, h), nil
}

func (p *Process) kill() error {
	return p.Signal(Kill)
}

// ProcessState stores information about a process, as reported by Wait.
type ProcessState struct {
	pid    int                // The process's id.
	status syscall.WaitStatus // System-dependent status info.
	rusage *syscall.Rusage
}

// Pid returns the process id of the exited process.
func (p *ProcessState) Pid() int {
	return p.pid
}

func (p *ProcessState) exited() bool {
	return p.status.Exited()
}

func (p *ProcessState) success() bool {
	return p.status.ExitStatus() == 0
}

func (p *ProcessState) sys() interface{} {
	return p.status
}

func (p *ProcessState) sysUsage() interface{} {
	return p.rusage
}

func (p *ProcessState) String() string {
	if p == nil {
		return "<nil>"
	}
	status := p.Sys().(syscall.WaitStatus)
	res := ""
	switch {
	case status.Exited():
		res = "exit status " + strconv.Itoa(status.ExitStatus())
	case status.Signaled():
		res = "signal: " + status.Signal().String()
	case status.Stopped():
		res = "stop signal: " + status.StopSignal().String()
		if status.StopSignal() == syscall.SIGTRAP && status.TrapCause() != 0 {
			res += " (trap " + strconv.Itoa(status.TrapCause()) + ")"
		}
	case status.Continued():
		res = "continued"
	}
	if status.CoreDump() {
		res += " (core dumped)"
	}
	return res
}

// ExitCode returns the exit code of the exited process, or -1
// if the process hasn't exited or was terminated by a signal.
func (p *ProcessState) ExitCode() int {
	// return -1 if the process hasn't started.
	if p == nil {
		return -1
	}
	return p.status.ExitStatus()
}

func (p *Process) wait() (ps *ProcessState, err error) {
	if p.Pid == -1 {
		return nil, syscall.EINVAL
	}

	// If we can block until Wait4 will succeed immediately, do so.
	ready, err := p.blockUntilWaitable()
	if err != nil {
		return nil, err
	}
	if ready {
		// Mark the process done now, before the call to Wait4,
		// so that Process.signal will not send a signal.
		p.setDone()
		// Acquire a write lock on sigMu to wait for any
		// active call to the signal method to complete.
		p.sigMu.Lock()
		p.sigMu.Unlock()
	}

	var status syscall.WaitStatus
	var rusage syscall.Rusage
	pid1, e := syscall.Wait4(p.Pid, &status, 0, &rusage)
	if e != nil {
		return nil, os.NewSyscallError("wait", e)
	}
	if pid1 != 0 {
		p.setDone()
	}
	ps = &ProcessState{
		pid:    pid1,
		status: status,
		rusage: &rusage,
	}
	return ps, nil
}

var errFinished = errors.New("os: process already finished")

func (p *Process) signal(sig os.Signal) error {
	if p.Pid == -1 {
		return errors.New("os: process already released")
	}
	if p.Pid == 0 {
		return errors.New("os: process not initialized")
	}
	p.sigMu.RLock()
	defer p.sigMu.RUnlock()
	if p.done() {
		return errFinished
	}
	s, ok := sig.(syscall.Signal)
	if !ok {
		return errors.New("os: unsupported signal type")
	}
	if e := syscall.Kill(p.Pid, s); e != nil {
		if e == syscall.ESRCH {
			return errFinished
		}
		return e
	}
	return nil
}

func (p *Process) release() error {
	// NOOP for unix.
	p.Pid = -1
	// no need for a finalizer anymore
	runtime.SetFinalizer(p, nil)
	return nil
}

func findProcess(pid int) (p *Process, err error) {
	// NOOP for unix.
	return newProcess(pid, 0), nil
}

func (p *ProcessState) userTime() time.Duration {
	return time.Duration(p.rusage.Utime.Nano()) * time.Nanosecond
}

func (p *ProcessState) systemTime() time.Duration {
	return time.Duration(p.rusage.Stime.Nano()) * time.Nanosecond
}

const _P_PID = 1

// blockUntilWaitable attempts to block until a call to p.Wait will
// succeed immediately, and reports whether it has done so.
// It does not actually call p.Wait.
func (p *Process) blockUntilWaitable() (bool, error) {
	// The waitid system call expects a pointer to a siginfo_t,
	// which is 128 bytes on all GNU/Linux systems.
	// On Darwin, it requires greater than or equal to 64 bytes
	// for darwin/{386,arm} and 104 bytes for darwin/amd64.
	// We don't care about the values it returns.
	var siginfo [16]uint64
	psig := &siginfo[0]
	_, _, e := syscall.Syscall6(syscall.SYS_WAITID, _P_PID, uintptr(p.Pid), uintptr(unsafe.Pointer(psig)), syscall.WEXITED|syscall.WNOWAIT, 0, 0)
	runtime.KeepAlive(p)
	if e != 0 {
		// waitid has been available since Linux 2.6.9, but
		// reportedly is not available in Ubuntu on Windows.
		// See issue 16610.
		if e == syscall.ENOSYS {
			return false, nil
		}
		return false, os.NewSyscallError("waitid", e)
	}
	return true, nil
}

func environForSysProcAttr(sys *scmpSyscall.SysProcAttr) ([]string, error) {
	return os.Environ(), nil
}
