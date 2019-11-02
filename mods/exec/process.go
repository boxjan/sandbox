package exec

import (
	"errors"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

type ProcessState struct {
	Pid    int                // The process's id.
	Status syscall.WaitStatus // System-dependent status info.
	Rusage *syscall.Rusage
}

// Process stores the information about a process created by StartProcess.
type Process struct {
	Pid    int
	handle uintptr      // handle is accessed atomically on Windows
	isdone uint32       // process has been successfully waited on, non zero if true
	sigMu  sync.RWMutex // avoid race between wait and signal
}

var errFinished = errors.New("os: process already finished")

func (p *ProcessState) UserTime() int {
	return int(p.Rusage.Utime.Nano() / 1e6)
}

func (p *ProcessState) SystemTime() int {
	return int(p.Rusage.Stime.Nano() / 1e6)
}

func (p *ProcessState) Memory() int {
	return 0
}

func (p *Process) release() error {
	// NOOP for unix.
	p.Pid = -1
	// no need for a finalizer anymore
	runtime.SetFinalizer(p, nil)
	return nil
}

func (p *Process) setDone() {
	atomic.StoreUint32(&p.isdone, 1)
}

func (p *Process) done() bool {
	return atomic.LoadUint32(&p.isdone) > 0
}

func (p *Process) Finished() bool {
	return p.done()
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
