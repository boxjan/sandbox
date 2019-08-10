package main

import (
	"context"
	"github.com/Boxjan/golib/logs"
	"github.com/containerd/cgroups"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sdibtacm/SandBox/mods/ScmpExec"
	scmpOs "github.com/sdibtacm/SandBox/mods/ScmpExec/os"
	scmpSyscall "github.com/sdibtacm/SandBox/mods/ScmpExec/syscall"
	"github.com/sdibtacm/SandBox/mods/seccomp"
	"os"
	"strconv"
	"syscall"
	"time"
)

type RunConfig struct {
	log   *logs.Logger
	limit *RuntimeLimit
	exec  *RuntimeExec
	io    *RuntimeIO

	pid              int
	process          *scmpOs.Process
	useCgroup        bool
	control          cgroups.Cgroup
	cgroupFolderPath string

	sigs chan os.Signal
}

type RuntimeExec struct {
	path string
	args []string
	env  []string
}

// Sandbox exit status
const (
	SANDBOX_RUN_SUCCESS int = iota
	SANDBOX_SYSTEM_FAIL
)

func (run *RunConfig) Run() *RuntimeResult {

	//run.sigs =  make(chan os.Signal, 1)
	//signal.Notify(run.sigs, syscall.SIGINT)
	//go func() {
	//	sig := <- run.sigs
	//	run.log.Info("Get signel: {}", sig)
	//	_ = syscall.Kill(-run.pid, syscall.SIGINT)
	//	time.Sleep(100 * time.Microsecond)
	//	//_ = run.process.Kill()
	//}()

	ctx := context.Background()
	var cancel context.CancelFunc
	//
	if run.limit.clockTime != 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(run.limit.clockTime)*time.Millisecond)
	}

	if run.limit.clockTime != 0 {
		defer cancel()
	}

	cmd := ScmpExec.CommandContext(ctx, run.exec.path, run.exec.args...)
	cmd.Env = append(run.exec.env)

	scmpFilter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		run.log.Error("seccomp init with error: {}", err)
	}

	cmd.SysProcAttr = &scmpSyscall.SysProcAttr{
		Cloneflags: getCloneFlagNull(),
		Setpgid:    true,
		//Ptrace: true,
		Scmp: scmpFilter,
	}

	cmd.Stdin = run.io.stdin
	cmd.Stdout = run.io.stdout
	cmd.Stderr = run.io.stderr

	run.log.Info("Ready to run command")
	err = cmd.Start()
	if nil != err {
		run.log.Error("run command meet error: {}", err)
		return &RuntimeResult{statusCode: SANDBOX_SYSTEM_FAIL}
	}

	startTime := time.Now()
	run.pid = cmd.Process.Pid
	run.process = cmd.Process
	//run.mountCgroup()

	//go run.memoryLimit()

	err = cmd.Wait()
	endTime := time.Now()

	if nil != err {
		run.log.Info("Process exit: {}", err)
	}

	if scmpFilter != nil {
		scmpFilter.Release()
	}

	//var usage syscall.Rusage
	//_ = syscall.Getrusage(syscall.RUSAGE_THREAD, &usage)
	res := getResult(cmd.ProcessState)
	res.clockTime = endTime.UnixNano() - startTime.UnixNano()
	//stat, err := run.control.Stat(cgroups.IgnoreNotExist)
	//
	//run.log.DebugF("%+v", stat)
	//run.log.Debug("{}", res.usage)
	run.log.Debug("{}", cmd.ProcessState.Sys())
	run.log.Debug("{}", cmd.ProcessState.ExitCode())
	run.unmountCgroup()
	return res
}

func getCloneFlagNull() uintptr {
	return 0
}

func getCloneFlagAll() uintptr {
	return syscall.CLONE_NEWIPC |
		syscall.CLONE_NEWNET |
		syscall.CLONE_NEWNS |
		syscall.CLONE_NEWPID |
		syscall.CLONE_NEWUSER |
		syscall.CLONE_NEWUTS
}

func (run *RunConfig) mountCgroup() {
	if os.Getuid() != 0 {
		run.log.Info("need run by root to use cgroup")
		run.useCgroup = false
		return
	}

	var err error
	cgroupFolderPath := "sandbox-" + strconv.FormatInt(time.Now().UnixNano(), 16)

	//config.cgroup , err = cgroups.New(cgroups.V1, cgroups.StaticPath(cgroupFolderPath), &specs.LinuxResources{})

	shares := uint64(100)
	swappiness := uint64(0)

	if run.limit.threadLimit < 1 {
		run.log.Warning("set thread count 8")
		run.limit.threadLimit = 8
	}

	run.control, err = cgroups.New(cgroups.V1, cgroups.StaticPath(cgroupFolderPath), &specs.LinuxResources{
		CPU: &specs.LinuxCPU{
			Shares: &shares,
		},
		Memory: &specs.LinuxMemory{
			Limit:      &run.limit.memory,
			Swappiness: &swappiness,
		},
		Pids: &specs.LinuxPids{
			Limit: run.limit.threadLimit,
		},
	})
	if err != nil {
		run.log.Warning("set cgroup with error: {}", err)
	}

	err = run.control.Add(cgroups.Process{Pid: run.pid})

	if err != nil {
		run.useCgroup = false
		run.log.Warning("cgroup will not use with error: {}", err)
	} else {
		run.useCgroup = true
	}

}

func (config *RunConfig) cgroupLimit() {
	if !config.useCgroup {
		return
	}

	//err := config.cgroup.Update(&specs.LinuxResources{
	//	Memory: &specs.LinuxMemory{
	//		Limit: &config.limit.memory,
	//	},
	//})
	//if err != nil {
	//	config.log.Warning("set cgroup with error: {}", err)
	//}

	err := config.control.Add(cgroups.Process{Pid: config.pid})

	if err != nil {
		config.log.Warning("set cgroup with error: {}", err)
	}

}

func (config *RunConfig) unmountCgroup() {
	if !config.useCgroup {
		return
	}
	config.log.Warning("cgroup will be delete")
	err := config.control.Delete()
	if err != nil {
		config.log.Warning("cgroup delete with error: {}", err)
	}

}
