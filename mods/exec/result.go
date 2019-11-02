package exec

import (
	"syscall"
)

type execResult struct {
	KernelTime uint
	UserTime   uint
	ClockTime  uint
	UsedMemory uint64
	ExitCode   int
	StatusCode syscall.WaitStatus
	Error      Error
}

func (c *Cmd) Result() (r *execResult) {

	if c.killHelper != nil && len(c.killHelper) == 0 {
		close(c.killHelper)
	}
	c.Logger.Debug("", c.startTimestamp.UnixNano())
	c.Logger.Debug("", c.endTimestamp.UnixNano())

	r = &execResult{
		KernelTime: uint(c.ProcessState.Rusage.Stime.Nano() / 1e6),
		UserTime:   uint(c.ProcessState.Rusage.Utime.Nano() / 1e6),
		ClockTime:  uint((c.endTimestamp.UnixNano() - c.startTimestamp.UnixNano()) / 1e6),
		UsedMemory: c.MemoryUsage / 1024,
		ExitCode:   c.ProcessState.Status.ExitStatus(),
		StatusCode: c.ProcessState.Status,
		Error:      Error{},
	}
	return
}
