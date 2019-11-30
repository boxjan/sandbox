// +build linux

package exec

import (
	"errors"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

// process helper
// because will use ptrace to trace process, the golang function have some insufficient

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

func (p *ProcessState) Exited() bool {
	return p.status.Exited()
}

func (p *ProcessState) Success() bool {
	return p.status.ExitStatus() == 0
}

func (p *ProcessState) Sys() syscall.WaitStatus {
	return p.status
}

func (p *ProcessState) SysUsage() *syscall.Rusage {
	return p.rusage
}

func (p *ProcessState) String() string {
	if p == nil {
		return "<nil>"
	}
	status := p.status
	res := ""
	switch {
	case status.Exited():
		res = "exit status " + itoa(status.ExitStatus())
	case status.Signaled():
		res = "signal: " + status.Signal().String()
	case status.Stopped():
		res = "stop signal: " + status.StopSignal().String()
		if status.StopSignal() == syscall.SIGTRAP && status.TrapCause() != 0 {
			res += " (trap " + itoa(status.TrapCause()) + ")"
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

// Process stores the information about a process created by StartProcess.
type Process struct {
	Pid    int
	handle uintptr      // handle is accessed atomically on Windows
	isdone uint32       // process has been successfully waited on, non zero if true
	sigMu  sync.RWMutex // avoid race between wait and signal
}

var errFinished = errors.New("os: process already finished")

func (p *Process) SignalGroup(sig os.Signal) error {
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
	if e := syscall.Kill(-p.Pid, s); e != nil {
		if e == syscall.ESRCH {
			return errFinished
		}
		return e
	}
	return nil
}

func (p *Process) Signal(sig os.Signal) error {
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

func newProcess(pid int, handle uintptr) *Process {
	p := &Process{Pid: pid, handle: handle}
	runtime.SetFinalizer(p, (*Process).Release)
	return p
}

func (p *Process) Release() error {
	// NOOP for unix.
	p.Pid = -1
	// no need for a finalizer anymore
	runtime.SetFinalizer(p, nil)
	return nil
}

func (p *Process) Kill() error {
	return p.Signal(os.Kill)
}

func (p *Process) KillGroup() error {
	return p.Signal(os.Kill)
}

func (p *Process) setDone() {
	atomic.StoreUint32(&p.isdone, 1)
}

func (p *Process) done() bool {
	return atomic.LoadUint32(&p.isdone) > 0
}

func (p *Process) SetDone() {
	p.setDone()
}

func (p *Process) Done() bool {
	return p.done()
}
