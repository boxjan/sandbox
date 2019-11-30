//+build linux

package exec

import (
	"github.com/sdibtacm/sandbox/exec/log"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

type ptrace struct {
	SyscallNo uint64
}

func (c *Cmd) wait() (ps *ProcessState, err error, pt *ptrace) {
	if c.Sys.Ptrace {
		return c.Process.ptraceWait()
	}

	ps, err = c.Process.Wait()
	return
}

func (p *Process) ptraceWait() (ps *ProcessState, err error, pt *ptrace) {
	log.GetLog().Debug("ptrace waiting up")

	var rusage syscall.Rusage
	var regs syscall.PtraceRegs
	var status syscall.WaitStatus
	var wpid int
	ps = &ProcessState{rusage: &rusage}

	wpid, err = syscall.Wait4(p.Pid, &status, 0, &rusage)
	log.GetLog().Debug("wait pid: {} status:{}, err: {}, the message is tell parent the child is ready", wpid, status, err)
	ps.status = status
	ps.pid = wpid

	// parent will trace only seccomp event
	_ = syscall.PtraceSetOptions(p.Pid, PTRACE_O_TRACESECCOMP)

	for {
		_ = syscall.PtraceCont(wpid, 0)
		wpid, err = syscall.Wait4(p.Pid, &status, 0, &rusage)
		ps.status = status
		ps.pid = wpid
		log.GetLog().Debug("wait pid: {} status:{}, err: {}", wpid, status, err)

		if err != nil {
			log.GetLog().Warning("wait4 error, error msg: {}", err)
			_ = p.KillGroup()
			return nil, err, nil
		}

		if status.Exited() || status.Signaled() {
			log.GetLog().Info("termination, exit status = {}, signal number = {}", status.ExitStatus(), status.Signaled())
			return
		}

		if status.Stopped() {
			log.GetLog().DebugF("child stopped, signal number=%d", status.StopSignal())
			if status.TrapCause() == PTRACE_EVENT_SECCOMP {
				log.GetLog().Debug("cache a seccomp event")
				_ = syscall.PtraceGetRegs(wpid, &regs)
				pt = &ptrace{SyscallNo: uint64(regs.Orig_rax)}
				_ = p.SignalGroup(syscall.SIGSYS)
			}
		} else {
			log.GetLog().Warning("don't know what happen, pid: {}, status: {}, err: {}", wpid, status, err)
			_ = p.KillGroup()
		}

	}
}

func (p *Process) Wait() (ps *ProcessState, err error) {
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
