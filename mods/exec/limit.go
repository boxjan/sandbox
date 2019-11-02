package exec

import (
	"github.com/sdibtacm/sandbox/mods/exec/limit"
	"time"
)

func (c *Cmd) Limit() (err error) {

	var limiter *limit.UsageTrace
	var usage *limit.Usage

	//if syscall.Getuid() == 0 {
	//	c.Logger.Debug("will up cgroup usage tracer")
	//	limiter, err = limit.NewCgroupUsageTrace(c.Process.Pid)
	//	if err != nil {
	//		return
	//	}
	//} else {
	c.Logger.Debug("will up pstree usage tracer")
	limiter, err = limit.NewPstreeUsageTrace(c.Process.Pid)
	if err != nil {
		return
	}
	//}

	go func() {
		for {
			select {
			case err := <-limiter.Err:
				{
					c.errch <- err
					break
				}
			default:
				usage = limiter.Tracer.Get()
				if time.Now().UnixNano()/1e5%5000 == 0 {
					c.Logger.Debug("", usage)
				}
				if c.Process.done() {
					break
				}
				c.MemoryUsage = usage.Memory
				if c.Resource.Memory < usage.Memory {
					c.Kill()
					c.killHelper <- Error{
						ErrorNum: MemoryExceedLimit,
					}
					break
				}

				if c.Resource.CpuTime < usage.KernelTime+usage.UserTime {
					c.killHelper <- Error{
						ErrorNum: CpuTimeExceedLimit,
					}
				}

				if c.Resource.Thread < uint64(usage.ThreadCount) {
					c.killHelper <- Error{
						ErrorNum: ThreadExceedLimit,
					}
				}
			}
		}

		err := limiter.Tracer.Destroy()
		if err != nil {
			c.errch <- err
		}
	}()

	return nil
}
