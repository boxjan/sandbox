// +build linux

package main

import (
	"errors"
	"fmt"
	"github.com/boxjan/golib/logs"
	"github.com/sdibtacm/sandbox/exec"
	"github.com/sdibtacm/sandbox/g"
	"github.com/sdibtacm/sandbox/units/helper"
	"github.com/sdibtacm/sandbox/units/version"
	"github.com/spf13/cobra"
	"os"
	"runtime"
	"strings"
	"time"
)

//
//import _ "net/http/pprof"
//
//func init() {
//	go func() {
//		log.Println(http.ListenAndServe("0.0.0.0:8080", nil))
//	}()
//}

var ErrNotFile = errors.New("not a file")
var ErrNotDir = errors.New("not a dir")

var stdinFile *os.File
var stdoutFile *os.File
var stderrFile *os.File

var cmd = &cobra.Command{
	Use:   "sandbox [flags] [COMMANDS]",
	Short: "Sandbox design for OnlineJudge",
	Long: `A sandbox design for OnlineJudge, but also can use for calc time, memory used.
Note: When you setting rlimit_*, will no check if the value is legal`,
	Example:       "sandbox -t 1000 -m 16m ./main.out",
	Args:          cobra.MinimumNArgs(1),
	Version:       version.Get(),
	SilenceErrors: true,
	SilenceUsage:  true,
	PreRun: func(cmd *cobra.Command, args []string) {
		Init()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if cmdShowSyscallHelp {

		}
		run(args[0], args...)
	},
}

var (
	cmdRlimit [17]uint64

	cmdLogPath    string
	cmdLogDebug   bool
	cmdLogVerbose bool

	cmdInputFilePath     string
	cmdOutputFilePath    string
	cmdErrOutputFilePath string

	cmdEnvs        string
	cmdUseHostEnvs bool
	cmdChdir       string
	cmdChroot      string

	cmdCpuTimeLimit   uint   // ms
	cmdClockTimeLimit uint   // ms
	cmdMemoryLimitStr string // byte
	cmdOutputLimitStr string // byte
	cmdThreadLimit    uint

	cmdSyscallLevel         int
	cmdSyscallHelper        string
	cmdShowSyscallHelp      bool
	cmdNoNewPrivs           bool
	cmdScmpDefaultAction    int
	cmdScmpBadSyscallAction int

	cmdUid   int
	cmdGid   int
	cmdUmask uint
)

func init() {
	initCmd()
}

func Init() {
	initLog()
}

func doBeforeExit() {
	g.CloseLog()

	if stdinFile != nil {
		stdinFile.Close()
	}
	if stdoutFile != nil {
		stdoutFile.Close()
	}
	if stderrFile != nil {
		stderrFile.Close()
	}

}

func initLog() {
	logLevel := "info"
	if cmdLogDebug {
		logLevel = "debug"
	}
	if cmdLogVerbose {
		logLevel = "trace"
	}

	var err error
	log := logs.NewLogger()
	if cmdLogPath != "console" {
		err = log.AddAdapter("file", logLevel, `{"filename":"`+cmdLogPath+`", "rotate": false}`)
	} else {
		err = log.AddAdapter("console", logLevel, ``)
	}
	if err != nil {
		_, _ = os.Stderr.Write([]byte("log add adapter fail with error: " + err.Error()))
		return
	}

	g.SetLog(log)
}

func main() {

	//runtime.GOMAXPROCS(1)
	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
	doBeforeExit()
}

func run(command string, args ...string) {
	var err error
	c := exec.Command(command, args[1:]...)
	err = handleCmd(c)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "handle option error: %v\n", err)
		return
	}

	runtime.LockOSThread()
	err = c.Start()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "handle option error: %v\n", err)
		return
	}
	go func() {
		t := time.NewTicker(1000 * time.Millisecond)
		defer t.Stop()
		for {
			<-t.C
			g.GetLog().DebugF("%+v", c.NowUsed())
		}
	}()
	err = c.Wait()
	if err != nil {
		return
	}
	runtime.UnlockOSThread()

	res := c.Result()
	fmt.Printf("%+v\n", res)
	return
}

func handleCmd(c *exec.Cmd) (err error) {
	cmdMemoryLimit := exec.BYTE_UNRESOURCE
	cmdOutputLimit := exec.BYTE_UNRESOURCE

	if cmdMemoryLimitStr != "" {
		cmdMemoryLimit = helper.StrToBytes(cmdMemoryLimitStr)
		if cmdMemoryLimit < exec.MIN_MEMORY_LIMIT {
			g.GetLog().Warning("memory limit is too small, will at least: {}", exec.MIN_MEMORY_LIMIT)
			cmdMemoryLimit = exec.MIN_MEMORY_LIMIT
		}
	}
	if cmdOutputLimitStr != "" {
		cmdOutputLimit = helper.StrToBytes(cmdOutputLimitStr)
		if cmdOutputLimit <= 0 {
			g.GetLog().Warning("output limit is less than 0, will not allow write anything")
		}
	}

	c.ResourceLimit.Memory = cmdMemoryLimit
	c.ResourceLimit.Output = cmdOutputLimit
	c.ResourceLimit.ClockTime = cmdClockTimeLimit
	c.ResourceLimit.CpuTime = cmdCpuTimeLimit
	c.ResourceLimit.Thread = cmdThreadLimit

	c.Sys = &exec.SysAttr{}
	for i := 0; i <= exec.RLIMIT_NLIMITS; i++ {
		c.Sys.RlimitList[i] = cmdRlimit[i]
	}

	if cmdUid > 0 || cmdGid > 0 || cmdUmask > 0 {
		if os.Getuid() == 0 {
			c.Sys.Credential = &exec.Credential{
				Uid:   cmdUid,
				Gid:   cmdGid,
				Umask: cmdUmask,
			}
		} else {
			g.GetLog().Warning("only root user can set uid,gid,umask")
		}
	}

	err1 := parseFile()
	if err1.Err != nil {
		return errors.New(err1.Error())
	}
	if stdinFile != nil {
		c.Stdin = stdinFile
	} else {
		c.Stdin = os.Stdin
	}
	if stdoutFile != nil {
		c.Stdout = stdoutFile
	} else {
		c.Stdout = os.Stdout
	}
	if stderrFile != nil {
		c.Stderr = stderrFile
	} else {
		c.Stderr = os.Stdout
	}

	if cmdChdir != "" {
		stat, err1 := os.Stat(cmdChdir)
		if err1 != nil {
			g.GetLog().Error("{} get stat with error: {}", cmdChdir, err1)
			return &FileError{Name: cmdChdir, Err: err}
		}
		if !stat.IsDir() {
			g.GetLog().Error("{} not a dir", cmdChdir)
			return &FileError{Name: cmdChdir, Err: ErrNotDir}
		}
		c.Chdir = cmdChdir
	}

	if cmdChroot != "" {
		stat, err1 := os.Stat(cmdChroot)
		if err1 != nil {
			g.GetLog().Error("{} get stat with error: {}", cmdChroot, err1)
			return &FileError{Name: cmdChroot, Err: err}
		}
		if !stat.IsDir() {
			g.GetLog().Error("{} not a dir", cmdChroot)
			return &FileError{Name: cmdChroot, Err: ErrNotDir}
		}
		c.Chroot = cmdChroot
	}

	if cmdEnvs != "" {
		envs := strings.Split(cmdEnvs, " ")
		c.Envs = envs
	}

	if cmdUseHostEnvs {
		c.Envs = append(c.Envs, os.Environ()...)
	}

	c.Syscall = &exec.SyscallLimit{}
	c.Syscall.Level = cmdSyscallLevel
	if cmdSyscallHelper != "" {
		c.Syscall.Helper = cmdSyscallHelper
		c.Syscall.Level = -1
	}
	if cmdNoNewPrivs {
		c.Sys.SetNoNewPrivs = true
	}
	c.Syscall.Action = cmdScmpDefaultAction << 8 & cmdScmpBadSyscallAction

	return
}

func parseFile() (err FileError) {
	if err = parseInput(); err.Err != nil {
		return
	}
	if err = parseOutputAndErrOutput(); err.Err != nil {
		return
	}
	return
}

func parseInput() (err FileError) {
	if cmdInputFilePath != "" {
		stat, err := os.Stat(cmdInputFilePath)
		if err != nil {
			g.GetLog().Error("{} file have some error when open: {}", cmdInputFilePath, err)
			return FileError{Name: cmdInputFilePath, Err: err}
		}
		if stat.IsDir() {
			g.GetLog().Error("{} not a file", cmdInputFilePath)
			return FileError{Name: cmdInputFilePath, Err: ErrNotFile}
		}
		stdinFile, err = os.OpenFile(cmdInputFilePath, os.O_WRONLY, 0644)
		if err != nil {
			g.GetLog().Error("{} open file meet error: {}", cmdInputFilePath, err)
			return FileError{Name: cmdInputFilePath, Err: err}
		}
	}
	return
}

func parseOutputAndErrOutput() (err FileError) {
	err1 := parseOutput(cmdOutputFilePath, &stdoutFile)
	err2 := parseOutput(cmdErrOutputFilePath, &stderrFile)
	if err1.Err != nil {
		return err1
	}
	if err2.Err != nil {
		return err2
	}
	return
}

func parseOutput(path string, filePtr **os.File) (err FileError) {
	var file *os.File
	if path != "" {
		stat, err := os.Stat(path)
		if err != nil {
			g.GetLog().Warning("{} file have some error when open: {}", path, err)
		}
		if stat != nil && stat.IsDir() {
			g.GetLog().Error("{} not a file", path)
			return FileError{Name: path, Err: ErrNotFile}
		}

		file, err = os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			g.GetLog().Error("{} open file meet error: {}", path, err)
			return FileError{Name: path, Err: err}
		}
	}
	*filePtr = file
	return
}

func initCmd() {
	flags := cmd.Flags()
	flags.SetInterspersed(false)
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_CPU], "rlimit-cpu", exec.RLIMIT_UNRESOURCE, "Set rlimit_cpu")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_FSIZE], "rlimit-fsize", exec.RLIMIT_UNRESOURCE, "Set rlimit_fsize")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_DATA], "rlimit-data", exec.RLIMIT_UNRESOURCE, "Set rlimit_data")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_STACK], "rlimit-sstack", exec.RLIMIT_UNRESOURCE, "Set rlimit_stack")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_CORE], "rlimit-core", exec.RLIMIT_UNRESOURCE, "Set rlimit_core")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_RSS], "rlimit-rss", exec.RLIMIT_UNRESOURCE, "Set rlimit_rss")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_NPROC], "rlimit-nproc", exec.RLIMIT_UNRESOURCE, "Set rlimit_nproc")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_NOFILE], "rlimit-nofile", exec.RLIMIT_UNRESOURCE, "Set rlimit_nofile")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_MEMLOCK], "rlimit-memlock", exec.RLIMIT_UNRESOURCE, "Set rlimit_memlock")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_AS], "rlimit-as", exec.RLIMIT_UNRESOURCE, "Set rlimit_as")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_LOCKS], "rlimit-locks", exec.RLIMIT_UNRESOURCE, "Set rlimit_locks")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_SIGPENDING], "rlimit-sigpending", exec.RLIMIT_UNRESOURCE, "Set rlimit_sigpending")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_MSGQUEUE], "rlimit-msgqueue", exec.RLIMIT_UNRESOURCE, "Set rlimit_msgqueue")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_NICE], "rlimit-nice", exec.RLIMIT_UNRESOURCE, "Set rlimit_nice")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_RTPRIO], "rlimit-rtprio", exec.RLIMIT_UNRESOURCE, "Set rlimit_rtprio")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_RTTIME], "rlimit-rttime", exec.RLIMIT_UNRESOURCE, "Set rlimit_rttime")
	flags.Uint64Var(&cmdRlimit[exec.RLIMIT_NLIMITS], "rlimit-nlimits", exec.RLIMIT_UNRESOURCE, "Set rlimit_nlimits")

	flags.StringVarP(&cmdLogPath, "log", "l", "console", "Log record set")
	flags.BoolVarP(&cmdLogDebug, "debug", "v", false, "Set log level debug")
	flags.BoolVar(&cmdLogVerbose, "verbose", false, "Record log verbose")

	flags.StringVarP(&cmdInputFilePath, "input-path", "i", "", "stdin redirect file")
	flags.StringVarP(&cmdOutputFilePath, "output-path", "o", "", "stdout redirect file")
	flags.StringVarP(&cmdErrOutputFilePath, "error-path", "x", "", "stderr redirect file")

	flags.StringVarP(&cmdEnvs, "env", "e", "", "Set exec environment, null is default")
	flags.BoolVar(&cmdUseHostEnvs, "use-host-env", false, "Use host env, will append from envs setting")
	flags.StringVar(&cmdChroot, "chroot", "", "Chroot to specified `path` before exec")
	flags.StringVar(&cmdChdir, "chdir", "", "Chdir to specified `path` after chroot")

	flags.UintVarP(&cmdCpuTimeLimit, "max-cpu-time", "b", exec.TIME_UNRESOURCE, "Limit clock time in micro seconds(ms)")
	flags.UintVarP(&cmdClockTimeLimit, "max-time", "t", exec.TIME_UNRESOURCE, "Limit cpu time in micro seconds(ms)")
	flags.StringVarP(&cmdMemoryLimitStr, "max-memory", "m", "", "Limit memory (+swap) usage. `bytes` supports common suffix like `k`, `m`, `g`\n")
	flags.StringVarP(&cmdOutputLimitStr, "max-output", "q", "", "Limit output. It will make a \"best  effort\" to enforce the limit but it is NOT accurate")
	flags.UintVarP(&cmdThreadLimit, "max-thread", "r", exec.SUGGEST_THREAD_LIMIT, "Limit thread.")

	flags.IntVarP(&cmdSyscallLevel, "syscall-limit-level", "p", 0, "Syscall limit level preset[0-7]")
	flags.StringVar(&cmdSyscallHelper, "syscall", "", "Apply a syscall filter")
	flags.BoolVar(&cmdShowSyscallHelp, "syscall-help", false, "show help about syscall")
	flags.BoolVar(&cmdNoNewPrivs, "no-new-privs", false, "Do not allow getting higher privileges using exec. This disables things like sudo, ping, etc. If you set syscall limit the flag will be true")
	flags.IntVar(&cmdScmpDefaultAction, "syscall-default-action", 0, "seccomp default action, only use when user have, 0: Deny, 1: Allow")
	flags.IntVar(&cmdScmpBadSyscallAction, "syscall-bad-syscall-action", 1, "seccomp default action, only use when user have, 0: kill, 1: Trace, 2: EPERM")

	flags.IntVarP(&cmdUid, "uid", "u", 0, "Set uid (`uid` must > 0). Only root can use this")
	flags.IntVarP(&cmdGid, "gid", "g", 0, "Set gid (`gid` must > 0). Only root can use this")
	flags.UintVar(&cmdUmask, "umask", 0, "Set Mask")
}
