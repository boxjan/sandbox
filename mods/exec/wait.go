package exec

import (
	"fmt"
	"syscall"
)

func (c *Cmd) wait() (state *ProcessState) {
	if c.usePtraceGetBadSyscall == true {
		state = c.ptraceWait()
	} else {
		state = c.normalWait()
	}
	_, _ = syscall.Wait4(c.Process.Pid, nil, 0, nil) // make sure no Z
	c.Process.setDone()
	return
}

func (c *Cmd) ptraceWait() (state *ProcessState) {
	c.Logger.Debug("ptrace waiting up")

	var rusage syscall.Rusage
	var regs syscall.PtraceRegs
	var status syscall.WaitStatus
	var wpid int
	state = &ProcessState{
		Pid:    c.Process.Pid,
		Rusage: &rusage,
	}

	wpid, err := syscall.Wait4(c.Process.Pid, &status, 0, &rusage)
	c.Logger.Debug("wait pid: {} status:{}, err: {}", wpid, status, err)

	_ = syscall.PtraceSetOptions(c.Process.Pid, PTRACE_O_TRACESECCOMP)

	for {
		_ = syscall.PtraceCont(wpid, 0)
		wpid, err := syscall.Wait4(c.Process.Pid, &status, 0, &rusage)
		c.Logger.Debug("wait pid: {} status:{}, err: {}", wpid, status, err)

		state.Status = status

		switch {
		case status.Exited():
			{
				c.Logger.DebugF("normal termination, exit status = %d", status.ExitStatus())
				return
			}
		case status.Signaled():
			{
				c.Logger.DebugF("abnormal termination, signal number = %d", status.Signal())
				if status.CoreDump() {
					c.Logger.DebugF("core file generated")
				}
				return
			}
		case status.Stopped():
			{
				c.Logger.InfoF("child stopped, signal number=%d", status.StopSignal())
				if status.TrapCause() == PTRACE_EVENT_SECCOMP {
					c.Logger.Info("cache a seccomp event")
					_ = syscall.PtraceGetRegs(wpid, &regs)
					c.Logger.DebugF("%+v", regs)
					c.Logger.Info("the process try to call {} with args", regs.Orig_rax)
					_ = syscall.Kill(-c.Process.Pid, syscall.SIGSYS) // kill all group
					c.killHelper <- Error{ErrorNum: BadSyscall, Helper: fmt.Sprintf("not allow syscall: %d", regs.Orig_rax)}
				}
			}
		}
	}

}

func (c *Cmd) normalWait() (state *ProcessState) {
	c.Logger.Debug("normal waiting up")

	var rusage syscall.Rusage
	var status syscall.WaitStatus
	state = &ProcessState{
		Pid:    c.Process.Pid,
		Rusage: &rusage,
	}

	for {
		wpid, err := syscall.Wait4(c.Process.Pid, &status, 0, &rusage)
		c.Logger.Debug("wait pid: {} status:{}, err: {}", wpid, status, err)

		state.Status = status

		switch {
		case status.Exited():
			{
				c.Logger.InfoF("normal termination, exit status = %d", status.ExitStatus())
				return
			}
		case status.Signaled():
			{
				c.Logger.InfoF("abnormal termination, signal number = %d", status.Signal())
				if status.CoreDump() {
					c.Logger.Info("core file generated")
				}
				return
			}
		case status.Stopped():
			{
				c.Logger.InfoF("child stopped, signal number=%d", status.StopSignal())
			}
		}
	}
}
