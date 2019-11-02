// +build linux

package main

import (
	"errors"
	"github.com/boxjan/golib/logs"
	"github.com/sdibtacm/sandbox/mods/exec"
	"os"
	"runtime"
	"syscall"
)

var errRunningNow = errors.New("running now")

func Command(name string, args ...string) (*SandboxRunConfig, error) {
	return CommandNotSameProgramName(name, name, args...)
}

func CommandNotSameProgramName(path string, name string, args ...string) (*SandboxRunConfig, error) {
	runner := GetDefaultSandboxRunConfig()

	execPath, err := findExecutable(path)
	if err != nil {
		return nil, err
	}

	runner.Lock()
	runner.exec.Path = execPath
	runner.exec.Args = append([]string{name}, args...)
	runner.Unlock()

	return runner, nil
}

func (s *SandboxRunConfig) SetLogger(logger *logs.Logger) {
	s.Lock()
	if logger != nil {
		s.Logger = logger
	}
	s.Unlock()
}

func (s *SandboxRunConfig) SetLimit(setting *exec.ResourceSetting) {
	s.Lock()
	s.Logger.Debug("User change resource limit before: {}", s.Resource)
	s.Resource = setting
	s.Logger.Debug("After: {}", s.Resource)
	s.Unlock()
}

func (s *SandboxRunConfig) SetLimitCount(clockTime uint, cpuTime uint64, memory uint64, outputSize uint64, threadCount uint64) {

	s.Lock()
	s.Logger.Debug("User change resource limit before: {}", s.Resource)
	s.Resource.ClockTime = clockTime
	s.Resource.CpuTime = cpuTime
	s.Resource.Memory = memory
	s.Resource.OutputSize = outputSize
	s.Resource.Thread = threadCount
	s.Logger.Debug("After: {}", s.Resource)
	s.Unlock()

	if clockTime > 2*60*1000 || cpuTime > 2*60*1000 { // 1min
		s.Logger.Warning("The time limit is too big, clock time: {}, cpu time: {}, maybe 1 min is enough",
			clockTime, cpuTime)
	}

	if memory > 4*1024*1024 { // 4G
		s.Logger.Warning("The memory limit set at {} is too big, maybe 2G is enough", memory)
	}

	if threadCount > 1024 {
		s.Logger.Warning("The thread limit set at {} too big, maybe 512 is enough", threadCount)
	}
}

func (s *SandboxRunConfig) SetEnv(env ...string) {
	s.exec.Env = env
}

func (s *SandboxRunConfig) SetDir(dir string) error {
	stat, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		s.exec.Dir = dir
		return nil
	} else {
		return errors.New(dir + " is not a dir")
	}
}

func (s *SandboxRunConfig) GetExec() *exec.ExecSetting {
	return s.exec
}

func (s *SandboxRunConfig) SetCredential(uid, gid int) error {

	if syscall.Getuid() != 0 {
		return errors.New("will no change uid or gid until you run by root")
	}

	s.Lock()
	s.Logger.Debug("User change Credential before: {}", s.Resource)
	if uid == 0 {
		s.Logger.Warning("it very danger run by root")
	}
	s.Credential.Uid = uid
	s.Credential.Gid = gid
	s.Logger.Debug("After: {}", s.Resource)
	s.Unlock()

	return nil
}

func (s *SandboxRunConfig) SetSyscall(level exec.RunPermissionLevel, usePtraceGetBadSyscall bool) error {
	s.Lock()
	s.Logger.Debug("User change syscall setting before: {}", s.Syscall)
	s.Syscall.UsePtraceGetBadSyscall = usePtraceGetBadSyscall
	s.Syscall.RunLevel = level
	s.Logger.Debug("After: {}", s.Syscall)
	s.Unlock()

	return nil

}

func (s *SandboxRunConfig) UseRootUserToRun() error {
	return s.SetCredential(0, 0)
}

func (s *SandboxRunConfig) Run() (result *SandboxRunResult) {
	result = getEmptySandboxRunResult()

	s.Lock()
	defer s.Unlock()

	if s.running {
		result.Error = exec.Error{ErrorNum: exec.SandboxError, Helper: "the process is already run"}
		return
	}
	s.running = true
	defer func(s *SandboxRunConfig) {
		s.running = false
	}(s)

	cmd, err := s.prepare()
	if err != nil {
		result.Error = exec.Error{ErrorNum: exec.SandboxError, Helper: err.Error()}
		return
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err = cmd.Start(); err != nil {
		result.Error = exec.Error{ErrorNum: exec.SandboxError, Helper: err.Error()}
		return
	}
	if err = cmd.Limit(); err != nil {
		result.Error = exec.Error{ErrorNum: exec.SandboxError, Helper: err.Error()}
		cmd.Kill()
		return
	}
	cmd.Wait()
	s.Logger.Debug("rusage: {}", cmd.ProcessState.Rusage)

	eR := cmd.Result()
	result.ClockTime = eR.ClockTime
	result.UserTime = eR.UserTime
	result.KernelTime = eR.KernelTime
	result.UsedMemory = eR.UsedMemory
	result.ExitCode = eR.ExitCode
	result.StatusCode = eR.StatusCode
	result.Error = eR.Error

	return
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
