package main

import scmpOs "github.com/sdibtacm/SandBox/mods/ScmpExec/os"

type RuntimeResult struct {
	kernelTime uint
	userTime   uint
	clockTime  int64
	usedMemory int64
	exitCode   int
	statusCode int
	usage      interface{}
}

func getResult(state *scmpOs.ProcessState) *RuntimeResult {
	return &RuntimeResult{
		kernelTime: uint(state.SystemTime()),
		userTime:   uint(state.UserTime()),
		usage:      state.SysUsage(),
		exitCode:   state.ExitCode(),
	}
}

//func getUsedMemory(state *os.ProcessState) int64 {
//	var rusage syscall.Rusage
//	rusage = (syscall.Rusage).state.SysUsage()
//	rusage
//}
