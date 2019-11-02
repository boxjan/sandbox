package limit

import (
	"fmt"
	"github.com/sdibtacm/sandbox/units/pstree"
	"sync/atomic"
	"time"
)

type PstreeUsageTrace struct {
	pid    int
	usage  Usage
	isdone uint32
}

func NewPstreeUsageTrace(pid int) (t *UsageTrace, err error) {
	tracer := PstreeUsageTrace{pid: pid}
	t0 := &UsageTrace{
		Tracer: &tracer,
		Err:    make(chan error),
	}
	_, err = pstree.New()
	if err != nil {
		return
	}
	t = t0

	go func() {
		for tracer.isdone == 0 {
			pt, err := pstree.New()
			if err != nil {
				t.Err <- err
				break
			}
			procs := pt.Procs

			usageCalc := Usage{
				KernelTime:  0,
				UserTime:    0,
				Memory:      0,
				ThreadCount: 0,
			}

			willCalc := make([]int, 0, 5)
			willCalc = append(willCalc, pid)
			for i := 0; i < len(willCalc); i++ {
				pid := willCalc[i]

				if len(procs[pid].Children) > 0 {
					willCalc = append(willCalc, procs[pid].Children...)
				}
				stat := procs[pid].Stat

				usageCalc.KernelTime += stat.Stime
				usageCalc.UserTime += stat.Utime
				usageCalc.Memory += stat.Vsize
				usageCalc.ThreadCount += stat.Nthreads

			}

			tracer.usage.Lock()
			tracer.usage.KernelTime = usageCalc.KernelTime
			tracer.usage.UserTime = usageCalc.UserTime
			tracer.usage.Memory = usageCalc.Memory
			tracer.usage.ThreadCount = usageCalc.ThreadCount
			tracer.usage.Unlock()

			if time.Now().UnixNano()%2e7 == 0 {
				fmt.Print(tracer.usage, "\n")
			}
		}
	}()

	return
}

func (t *PstreeUsageTrace) Destroy() (err error) {
	atomic.StoreUint32(&t.isdone, 1)
	return nil
}

func (t *PstreeUsageTrace) Get() *Usage {
	t.usage.RLock()
	defer t.usage.RUnlock()
	return &t.usage
}
