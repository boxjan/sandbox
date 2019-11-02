// +build linux

// this mod is use Seccomp to limit a syscall
// here also have limit to limit memory use and thread
// although we hope use Cgroup to limit, but it need root and /sys/fs/cgroup privilege
// so if a use run no by root, we will use /proc/{pid}/stats to calc and checkout
// Most of the code is copied from lib, so it will have little comment

package exec

import (
	"context"
	"github.com/boxjan/golib/logs"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

type ResourceLimit struct {
	CpuTime uint64
	Memory  uint64
	Thread  uint64
}

type Cmd struct {
	Logger *logs.Logger

	Exec         *ExecAttr
	Process      *Process
	ProcessState *ProcessState
	Resource     *ResourceLimit
	ctx          context.Context
	ctxCancel    context.CancelFunc

	usePtraceGetBadSyscall bool
	followResourceUsage    bool

	closeAfterStart []io.Closer
	closeAfterWait  []io.Closer
	goroutine       []func() error
	errch           chan error // one send per goroutine
	waitDone        chan struct{}

	killHelper chan Error
	clockTime  time.Duration

	startTimestamp time.Time
	endTimestamp   time.Time
	MemoryUsage    uint64
}

func (c *Cmd) Start() error {

	var err error
	c.Process, err = c.forkAndExec()
	if err != nil {
		c.closeDescriptors(c.closeAfterStart)
		c.closeDescriptors(c.closeAfterWait)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.clockTime)
	c.ctx = ctx
	c.ctxCancel = cancel
	c.startTimestamp = time.Now()
	c.closeDescriptors(c.closeAfterStart)
	c.waitDone = make(chan struct{})

	c.errch = make(chan error, len(c.goroutine))
	for _, fn := range c.goroutine {
		go func(fn func() error) {
			c.errch <- fn()
		}(fn)
	}

	go func() {
		err := <-c.errch
		c.Logger.Warning("{}", err)
	}()

	return nil
}

func (c *Cmd) Wait() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-c.waitDone:
				{
					c.ctxCancel()
					break
				}
			case sig := <-sigs:
				_ = c.Process.Signal(sig)
			case <-c.ctx.Done():
				{
					c.Logger.Warning("Clock Timeout")
					_ = c.Process.Signal(os.Kill)
					c.killHelper <- Error{ErrorNum: ClockTimeExceedLimit}
				}
			}
		}
	}()

	c.ProcessState = c.wait()
	c.endTimestamp = time.Now()
	signal.Stop(sigs)
	if c.waitDone != nil {
		close(c.waitDone)
	}
	close(sigs)
	close(c.errch)
	c.closeDescriptors(c.closeAfterWait)
}

func (c *Cmd) closeDescriptors(closers []io.Closer) {
	for _, fd := range closers {
		_ = fd.Close()
	}
}
func (c *Cmd) Kill() {
	_ = c.Process.Signal(os.Kill)
}

func (c *Cmd) forkAndExec() (process *Process, err error) {
	var p [2]int
	var n int
	var err1 syscall.Errno
	var wstatus syscall.WaitStatus

	var r1 uintptr
	var locked bool
	var pid int

	ForkLock.Lock()
	if err = forkExecPipe(p[:]); err != nil {
		goto error
	}
	if r1, err1, _, locked = forkAndExecInChild(c.Exec, p[1]); locked {
		runtimeAfterFork()
	}
	if err1 != 0 {
		err = syscall.Errno(err1)
		goto error
	}
	ForkLock.Unlock()
	pid = int(r1)

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
		return nil, err
	}

	process = &Process{Pid: pid, handle: 0}
	runtime.SetFinalizer(process, (*Process).release)
	return

error:
	if p[0] >= 0 {
		_ = syscall.Close(p[0])
		_ = syscall.Close(p[1])
	}
	ForkLock.Unlock()
	return nil, err
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
