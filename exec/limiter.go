//+build linux

package exec

/*
#include <unistd.h>
int tickPreSec() {
	return sysconf(_SC_CLK_TCK);
}
*/
import "C"

import (
	"github.com/sdibtacm/sandbox/exec/log"
	"github.com/sdibtacm/sandbox/units/pstree"
	"time"
)

var tickPerSec int
var kernelTimeMod uint

func init() {
	tickPerSec = int(C.tickPreSec())
	kernelTimeMod = uint(1000 / float64(tickPerSec))
}

func (c *Cmd) limiter() {
	var used Resource
	for !c.Process.Done() {
		_ = calcUsed(&used, c.Process.Pid)
		//time.Sleep(5 * time.Microsecond)
		c.resourceStats.ClockTime = uint((time.Now().UnixNano() - c.startTimestamp.UnixNano()) / 1e6)
		c.resourceStats.Thread = used.Thread
		c.resourceStats.Memory = used.Memory
		c.resourceStats.CpuTime = used.CpuTime
		used.Memory = 0
		used.Thread = 0
		used.CpuTime = 0
		go c.updateMaxResource()
		go c.checkResource()
	}
}

func (c *Cmd) checkResource() {
	if c.ResourceLimit.Memory != BYTE_UNRESOURCE && c.resourceMaxStats.Memory > c.ResourceLimit.Memory {
		_ = c.Process.KillGroup()
	}
	if c.resourceMaxStats.Thread > c.ResourceLimit.Thread {
		_ = c.Process.KillGroup()
	}
	if c.ResourceLimit.CpuTime != TIME_UNRESOURCE && c.resourceMaxStats.CpuTime > c.ResourceLimit.CpuTime {
		_ = c.Process.KillGroup()
	}
}

func (c *Cmd) updateMaxResource() {
	if c.resourceMaxStats.Memory < c.resourceStats.Memory {
		c.resourceMaxStats.Memory = c.resourceStats.Memory
	}
	if c.resourceMaxStats.Thread < c.resourceStats.Thread {
		c.resourceMaxStats.Thread = c.resourceStats.Thread
	}
	if c.resourceMaxStats.CpuTime < c.resourceStats.CpuTime {
		c.resourceMaxStats.CpuTime = c.resourceStats.CpuTime
	}
}

func calcUsed(r *Resource, rootPid int) error {
	pt, err := pstree.New()
	if err != nil {
		log.GetLog().Warning("pstree scan error: {}", err)
		return err
	}
	procs := pt.Procs
	pids := []int{rootPid}
	//r.CpuTime = uint(procs[rootPid].Stat.Stime + procs[rootPid].Stat.Utime + procs[rootPid].Stat.Cstime + procs[rootPid].Stat.Cutime) * kernelTimeMod
	for i := 0; i < len(pids); i++ {
		pid := pids[i]
		if len(procs[pid].Children) > 0 {
			pids = append(pids, procs[pid].Children...)
		}
		r.CpuTime += uint(procs[pid].Stat.Stime+procs[pid].Stat.Utime) * kernelTimeMod
		r.Memory += procs[pid].Stat.Vsize
		r.Thread += uint(procs[pid].Stat.Nthreads)
	}
	return nil
}
