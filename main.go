package main

import (
	"github.com/boxjan/golib/logs"
	"os"
	"runtime"
	"syscall"
)

func main() {
	runtime.GOMAXPROCS(1)
	log := logs.NewLogger()
	err := log.AddAdapter("console", "trace", `{"filename":"sandbox.log", "rotate": false}`)
	if err != nil {
		_, _ = os.Stderr.Write([]byte("log add adapter fail with error: " + err.Error()))
	}

	//runner, err := Command("stress", "--vm-bytes", "256m", "--vm-keep", "-m", "2", "--timeout", "10s")
	runner, err := Command("stress", "--vm-bytes", "256m", "--vm-keep", "-m", "2")
	//runner, err := Command("ping", "-c", "5", "baidu.com")
	//runner, err := Command("sh")
	//runner, err := Command("/home/hzj/fork.out")

	if err != nil {
		panic(err)
	}
	//if syscall.Getuid() == 0 {
	//	_ = runner.UseRootUserToRun()
	//}
	runner.SetLogger(log)
	runner.SetEnv(syscall.Environ()...)
	err = runner.SetDir("/home")
	if err != nil {
		panic(err)
	}
	runner.SetLimit(GetUnlimitResourceSetting())

	result := runner.Run()

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
