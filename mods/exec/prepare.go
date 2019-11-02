package exec

import (
	"errors"
	"github.com/sdibtacm/sandbox/mods/seccmopFilter"
	"io"
	"os"
	"syscall"
	"time"
	"unsafe"
)

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

func (c *Cmd) PrepareExec(s *ExecSetting) error {
	if c.Exec == nil {
		c.Exec = &ExecAttr{}
	}

	if len(s.Path) > 0 {
		path, err := syscall.BytePtrFromString(s.Path)
		if err != nil {
			return err
		}
		c.Exec.Path = path
	} else {
		return errors.New("empty exec path")
	}

	if len(s.Args) > 0 {
		args, err := syscall.SlicePtrFromStrings(s.Args)
		if err != nil {
			return err
		}
		c.Exec.Args = args
	} else {
		c.Logger.Warning("args is empty")
	}

	if len(s.Env) > 0 {
		envs, err := syscall.SlicePtrFromStrings(s.Env)
		if err != nil {
			return err
		}
		c.Exec.Env = envs
	} else {
		c.Logger.Warning("env is empty")
	}

	if s.Dir != "" {
		chdir, err := syscall.BytePtrFromString(s.Dir)
		if err != nil {
			return err
		}
		c.Exec.Chdir = chdir
	} else {
		c.Logger.Warning("chdir is empty")
	}

	return nil
}

func (c *Cmd) PrepareIo(s *IoSetting) (err error) {
	if c.Exec == nil {
		c.Exec = &ExecAttr{}
	}
	var stdinFd, stdoutFd, stderrFd uintptr
	if stdinFd, err = c.stdin(s.Stdin); err != nil {
		return
	}
	if stdoutFd, err = c.stdout(s.Stdout); err != nil {
		return
	}
	if stderrFd, err = c.stderr(s.Stdout); err != nil {
		return
	}

	c.Exec.Files = []uintptr{stdinFd, stdoutFd, stderrFd}
	return
}

func (c *Cmd) stdin(r io.Reader) (fd uintptr, err error) {
	var f *os.File
	var ok bool

	if r == nil {
		f, err = os.Open(os.DevNull)
		if err != nil {
			return
		}
		fd = f.Fd()
		c.closeAfterStart = append(c.closeAfterStart, f)
		return
	}

	// give me a file
	if f, ok = r.(*os.File); ok {
		fd = f.Fd()
	}

	// give me other
	pr, pw, err := os.Pipe()
	if err != nil {
		return
	}
	c.closeAfterStart = append(c.closeAfterStart, pr)
	c.closeAfterWait = append(c.closeAfterWait, pw)
	c.goroutine = append(c.goroutine, func() error {
		_, err := io.Copy(pw, r)
		if skip := skipStdinCopyError; skip != nil && skip(err) {
			err = nil
		}
		if err1 := pw.Close(); err == nil {
			err = err1
		}
		return err
	})
	fd = pr.Fd()
	return
}

func (c *Cmd) stdout(w io.Writer) (fd uintptr, err error) {
	return c.writerDescriptor(w)
}

func (c *Cmd) stderr(w io.Writer) (fd uintptr, err error) {
	return c.writerDescriptor(w)
}

func (c *Cmd) writerDescriptor(w io.Writer) (fd uintptr, err error) {
	var f *os.File
	var ok bool

	if w == nil {
		f, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		c.closeAfterStart = append(c.closeAfterStart, f)
		fd = f.Fd()
		return
	}

	if f, ok = w.(*os.File); ok {
		fd = f.Fd()
		return
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
	fd = pw.Fd()
	return
}

func (c *Cmd) PrepareSyscall(s *SyscallSetting) (err error) {
	if c.Exec == nil {
		c.Exec = &ExecAttr{}
	}
	c.Exec.Ptrace = s.UsePtraceGetBadSyscall
	c.usePtraceGetBadSyscall = s.UsePtraceGetBadSyscall
	c.Exec.ScmpBpf, c.Exec.SetNoNewPrivs, err = seccmopFilter.GetFilterByLevel(
		int8(s.RunLevel), s.UsePtraceGetBadSyscall, unsafe.Pointer(c.Exec.Path))
	return
}

func (c *Cmd) PrepareLimit(s *ResourceSetting) (err error) {

	if s.OutputSize != RESOURCE_UNLIMIT {
		c.Exec.OutputSizeRlimit = &syscall.Rlimit{
			Cur: uint64(s.OutputSize),
			Max: uint64(s.OutputSize),
		}
	}
	c.Resource = &ResourceLimit{
		CpuTime: uint64(s.CpuTime),       // millisecond
		Memory:  uint64(s.Memory) * 1024, // bytes now
		Thread:  uint64(s.Thread),
	}
	c.clockTime = time.Duration(s.ClockTime) * time.Millisecond
	return
}

func (c *Cmd) PrepareCredential(s *Credential) (err error) {
	if s == nil {
		return nil
	}
	if c.Exec == nil {
		c.Exec = &ExecAttr{}
	}

	if syscall.Getuid() != s.Uid {
		c.Exec.Uid = s.Uid
		c.Exec.SetUid = true
	}

	if syscall.Getgid() != s.Gid {
		c.Exec.Gid = s.Gid
		c.Exec.SetGid = true
	}

	return
}
