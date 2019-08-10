package main

import (
	"github.com/Boxjan/golib/logs"
	"os"
	"syscall"
)

func main() {
	log := logs.NewLogger()
	err := log.AddAdapter("console", "trace", `{"filename":"sandbox.log", "rotate": false}`)
	if err != nil {
		_, _ = os.Stderr.Write([]byte("log add adapter fail with error" + err.Error()))
	}

	limit := RuntimeLimit{clockTime: 0, memory: 2048 * 1024 * 1024, threadLimit: 8}
	io := RuntimeIO{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}
	//exec := RuntimeExec{path: `/home/hzj/fork.out`, args: []string{""}, env: syscall.Environ()} // thread
	//exec := RuntimeExec{path: `/home/hzj/clone.out`, args: []string{""}, env: syscall.Environ()} // process
	//exec := RuntimeExec{path: `stress`, args: []string{"--vm-bytes", "256m", "--vm-keep", "-m", "4", "--timeout", "3s"}, env: syscall.Environ()} // Memory
	exec := RuntimeExec{path: `ping`, args: []string{"-c", "5", "192.168.2.1"}, env: syscall.Environ()} // NETWORK
	//exec := RuntimeExec{path: `bash`, args: []string{}, env: syscall.Environ()} // shell

	config := RunConfig{
		limit: &limit,
		exec:  &exec,
		io:    &io,
		log:   log,
	}

	result := config.Run()

	log.Debug("{}", result)

	//shares := uint64(100)
	//control, err := cgroups.New(cgroups.V1, cgroups.StaticPath("sandbox-test"), &specs.LinuxResources{
	//	CPU: &specs.LinuxCPU{
	//	Shares: &shares,
	//	},
	//})
	//stats, err := control.Stat(cgroups.IgnoreNotExist)
	//if err != nil {
	//	log.Debug("{}", err)
	//}
	//log.Debug("{}", stats)
}
