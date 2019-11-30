//+build linux

package exec

import "C"
import (
	"context"
	"errors"
	"github.com/boxjan/golib/logs"
	"github.com/sdibtacm/sandbox/exec/log"
	"github.com/sdibtacm/sandbox/exec/scmpFilter"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

// skipStdinCopyError optionally specifies a function which reports
// whether the provided stdin copy error should be ignored.
// It is non-nil everywhere but Plan 9, which lacks EPIPE. See exec_posix.go.
var skipStdinCopyError func(error) bool

func init() {
	skipStdinCopyError = func(err error) bool {
		// Ignore EPIPE errors copying to stdin if the program
		// completed successfully otherwise.
		// See Issue 9173.
		pe, ok := err.(*os.PathError)
		return ok &&
			pe.Op == "write" && pe.Path == "|1" &&
			pe.Err == syscall.EPIPE
	}
}

type Cmd struct {
	Path   string   // run command path
	Args   []string // run command args
	Envs   []string
	Chroot string
	Chdir  string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	ResourceLimit    Resource
	resourceMaxStats Resource
	resourceStats    Resource
	Sys              *SysAttr
	Syscall          *SyscallLimit
	Process          *Process
	ProcessState     *ProcessState

	finished        bool // when Wait was called
	ctx             context.Context
	ctxCancel       context.CancelFunc
	closeAfterStart []io.Closer
	closeAfterWait  []io.Closer
	childFiles      []*os.File
	goroutine       []func() error
	errch           chan error // one send per goroutine
	sigchan         chan os.Signal
	waitDone        chan struct{}
	pt              *ptrace

	startTimestamp time.Time
	endTimestamp   time.Time
}

type Result struct {
	CpuTime    uint
	ClockTime  uint
	MemoryUsed uint64
	ExitStatus syscall.WaitStatus
	ExitCode   int
	Exceed     int
	HelpStr    string
}

type Resource struct {
	CpuTime   uint
	ClockTime uint
	Memory    uint64
	Output    uint64
	Thread    uint
}

type SyscallLimit struct {
	Level  int
	Action int
	Helper string
}

func SetLogger(logger *logs.Logger) {
	log.SetLog(logger)
}

func findExecutable(name string) (string, error) {
	if err := executable(name); err == nil {
		return name, nil
	}
	if lp, err := LookPath(name); err != nil {
		return "", err
	} else {
		return lp, nil
	}
}

func Command(name string, args ...string) *Cmd {
	return CommandNotSameProgramName(name, name, args...)
}

func CommandNotSameProgramName(command string, name string, args ...string) *Cmd {
	cmd := &Cmd{}
	cmd.Path = command
	cmd.Args = append([]string{name}, args...)
	return cmd
}

func (c *Cmd) envv() []string {
	if c.Envs != nil {
		return c.Envs
	}
	return os.Environ()
}

func (c *Cmd) argv() []string {
	if len(c.Args) > 0 {
		return c.Args
	}
	return []string{c.Path}
}

// interfaceEqual protects against panics from doing equality tests on
// two interfaces with non-comparable underlying types.
func interfaceEqual(a, b interface{}) bool {
	defer func() {
		recover()
	}()
	return a == b
}

func (c *Cmd) stdin() (f *os.File, err error) {
	if c.Stdin == nil {
		f, err = os.Open(os.DevNull)
		if err != nil {
			return
		}
		c.closeAfterStart = append(c.closeAfterStart, f)
		return
	}

	if f, ok := c.Stdin.(*os.File); ok {
		return f, nil
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return
	}

	c.closeAfterStart = append(c.closeAfterStart, pr)
	c.closeAfterWait = append(c.closeAfterWait, pw)
	c.goroutine = append(c.goroutine, func() error {
		_, err := io.Copy(pw, c.Stdin)
		if skip := skipStdinCopyError; skip != nil && skip(err) {
			err = nil
		}
		if err1 := pw.Close(); err == nil {
			err = err1
		}
		return err
	})
	return pr, nil
}

func (c *Cmd) stdout() (f *os.File, err error) {
	return c.writerDescriptor(c.Stdout)
}

func (c *Cmd) stderr() (f *os.File, err error) {
	if c.Stderr != nil && interfaceEqual(c.Stderr, c.Stdout) {
		return c.childFiles[1], nil
	}
	return c.writerDescriptor(c.Stderr)
}

func (c *Cmd) writerDescriptor(w io.Writer) (f *os.File, err error) {
	if w == nil {
		f, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		c.closeAfterStart = append(c.closeAfterStart, f)
		return
	}

	if f, ok := w.(*os.File); ok {
		return f, nil
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return
	}

	c.closeAfterStart = append(c.closeAfterStart, pw)
	c.closeAfterWait = append(c.closeAfterWait, pr)
	c.goroutine = append(c.goroutine, func() error {
		_, err := io.Copy(w, pr)
		pr.Close() // in case io.Copy stopped due to write error
		return err
	})
	return pw, nil
}

func (c *Cmd) closeDescriptors(closers []io.Closer) {
	for _, fd := range closers {
		fd.Close()
	}
}

func (c *Cmd) Start() error {
	execPath, err := findExecutable(c.Path)
	if err != nil {
		log.GetLog().Warning("{} can not exec", c.Path)
		return err
	}
	c.Path = execPath

	if c.Process != nil {
		return errors.New("exec: already started")
	}

	c.childFiles = make([]*os.File, 0, 3)
	type F func(*Cmd) (*os.File, error)
	for _, setupFd := range []F{(*Cmd).stdin, (*Cmd).stdout, (*Cmd).stderr} {
		fd, err := setupFd(c)
		if err != nil {
			log.GetLog().Error("fail to get os.File struct with error: {}", err)
			c.closeDescriptors(c.closeAfterStart)
			c.closeDescriptors(c.closeAfterWait)
			return err
		}
		c.childFiles = append(c.childFiles, fd)
	}

	// set cpu time and clock time
	if c.Sys.RlimitList[RLIMIT_CPU] == RLIMIT_UNRESOURCE && c.ResourceLimit.CpuTime != TIME_UNRESOURCE {
		log.GetLog().Debug("cpu time limit will be set {}s", uint64(c.ResourceLimit.CpuTime/1000+1))
		c.Sys.RlimitList[RLIMIT_CPU] = uint64(c.ResourceLimit.CpuTime/1000 + 1)
		if c.ResourceLimit.ClockTime == TIME_UNRESOURCE {
			log.GetLog().Info("Have set cpu time, but not set clock time, clock time will be set {}ms, to keep safe", c.ResourceLimit.CpuTime*10)
			if uint64(c.ResourceLimit.CpuTime*10) > uint64(MAX_TIME) {
				c.ResourceLimit.ClockTime = MAX_TIME
			} else {
				c.ResourceLimit.ClockTime = c.ResourceLimit.CpuTime * 10
			}
		}
		if c.ResourceLimit.ClockTime < c.ResourceLimit.CpuTime {
			log.GetLog().Info("clock time limit is small than cpu time, will change clock time to", c.ResourceLimit.CpuTime+1)
			if uint64(c.ResourceLimit.CpuTime+1) > uint64(MAX_TIME) {
				c.ResourceLimit.ClockTime = MAX_TIME
			} else {
				c.ResourceLimit.ClockTime = c.ResourceLimit.CpuTime
			}
		}
	}

	if c.Sys.RlimitList[RLIMIT_FSIZE] == RLIMIT_UNRESOURCE && c.ResourceLimit.Output != BYTE_UNRESOURCE {
		c.Sys.RlimitList[RLIMIT_FSIZE] = c.ResourceLimit.Output + 1
	}

	log.GetLog().Debug("will start process")
	c.Process, err = c.startProcess()
	if err != nil {
		log.GetLog().Error("start process fail with error: {}", err)
		c.closeDescriptors(c.closeAfterStart)
		c.closeDescriptors(c.closeAfterWait)
		return err
	}

	go c.SentSig()
	c.startTimestamp = time.Now()
	if c.ResourceLimit.ClockTime != TIME_UNRESOURCE {
		c.ctx, c.ctxCancel = context.WithTimeout(context.Background(), time.Duration(c.ResourceLimit.ClockTime)*time.Millisecond)
	}
	c.closeDescriptors(c.closeAfterStart)

	go c.limiter()
	// Don't allocate the channel unless there are goroutines to fire.
	if len(c.goroutine) > 0 {
		c.errch = make(chan error, len(c.goroutine))
		for _, fn := range c.goroutine {
			go func(fn func() error) {
				c.errch <- fn()
			}(fn)
		}
	}

	if c.ctx != nil {
		c.waitDone = make(chan struct{})
		go func() {
			select {
			case <-c.ctx.Done():
				_ = c.Process.KillGroup()
			case <-c.waitDone:
			}
		}()
	}

	return nil
}

func (c *Cmd) NowUsed() Resource {
	return c.resourceStats
}

func (c *Cmd) Wait() error {
	if c.Process == nil {
		return errors.New("exec: not started")
	}
	if c.finished {
		return errors.New("exec: Wait was already called")
	}
	c.finished = true

	state, err, pt := c.wait()
	c.endTimestamp = time.Now()
	if c.ctxCancel != nil {
		c.ctxCancel()
	}

	if err != nil {
		return err
	}
	if c.waitDone != nil {
		close(c.waitDone)
	}
	c.ProcessState = state
	c.pt = pt

	var copyError error
	for range c.goroutine {
		if err := <-c.errch; err != nil && copyError == nil {
			copyError = err
		}
	}

	c.closeDescriptors(c.closeAfterWait)

	if err != nil {
		return err
	}

	return copyError
}

func (c *Cmd) Result() *Result {

	r := &Result{
		CpuTime:    uint(c.ProcessState.rusage.Utime.Nano()+c.ProcessState.rusage.Stime.Nano()) / 1e6,
		ClockTime:  uint(c.endTimestamp.UnixNano()-c.startTimestamp.UnixNano()) / 1e6,
		MemoryUsed: c.resourceStats.Memory,
		ExitStatus: c.ProcessState.status,
		ExitCode:   c.ProcessState.ExitCode(),
	}

	return r
}

func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Cmd) startProcess() (*Process, error) {

	path0, err := syscall.BytePtrFromString(c.Path)
	if err != nil {
		return nil, err
	}
	argsp, err := syscall.SlicePtrFromStrings(c.Args)
	if err != nil {
		return nil, err
	}
	if len(c.Envs) == 0 {
		c.Envs = os.Environ()
	}
	envsp, err := syscall.SlicePtrFromStrings(c.Envs)
	if err != nil {
		return nil, err
	}
	var chroot *byte
	if c.Chroot != "" {
		chroot, err = syscall.BytePtrFromString(c.Chroot)
		if err != nil {
			return nil, err
		}
	}
	var chdir *byte
	if c.Chdir != "" {
		chdir, err = syscall.BytePtrFromString(c.Chdir)
		if err != nil {
			return nil, err
		}
	}
	var attr *SysAttr
	if c.Sys == nil {
		attr = &SysAttr{}
	} else {
		attr = c.Sys
	}

	if len(c.childFiles) != 0 {
		attr.Files = make([]uintptr, 0, len(c.childFiles))
		for _, f := range c.childFiles {
			attr.Files = append(attr.Files, f.Fd())
		}
	}

	if c.Syscall != nil && c.Syscall.Helper != "" && c.Syscall.Level != 0 {
		scmpHelper := &scmpFilter.ScmpFilterLoadHelper{ExecvePathPointer: unsafe.Pointer(path0), Action: scmpFilter.ScmpAction(c.Syscall.Action)}
		if c.Syscall.Helper != "" {
			scmpHelper.LrunScmpFilter = c.Syscall.Helper
			scmpHelper.Level = -1
		} else {
			scmpHelper.Level = c.Syscall.Level
		}
		filter, err := scmpFilter.GetScmpFilter(scmpHelper)
		if err != nil {
			return nil, err
		}
		attr.Bpf = filter.BPF
		attr.Ptrace = filter.SetPrivs
	}

	pid, err := forkExec(path0, argsp, envsp, chroot, chdir, attr)
	if err != nil {
		log.GetLog().Error("exec fail with error: {}", err.Error())
		return nil, errors.New(err.Error())
	}
	return newProcess(pid, 0), nil
}

func (c *Cmd) SentSig() {
	c.sigchan = make(chan os.Signal)
	signal.Notify(c.sigchan, syscall.SIGINT, syscall.SIGTERM)

	for sig, ok := <-c.sigchan; (!c.Process.Done()) && ok; {
		err := c.Process.Signal(sig)
		if err != nil {
			log.GetLog().Warning("sent sig meet error: {}", err)
		}
	}
	signal.Stop(c.sigchan)
	close(c.sigchan)
	c.sigchan = nil
}
