package limit

import "sync"

type Usage struct {
	sync.RWMutex
	KernelTime  uint64 // ms
	UserTime    uint64 // ms
	Memory      uint64 // byte
	ThreadCount int64
}

type UsageTrace struct {
	Tracer UsageInter
	Err    chan error
}

type UsageInter interface {
	Get() *Usage
	Destroy() error
}
