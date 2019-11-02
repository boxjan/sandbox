package exec

const (
	AllIsWell int = iota
	CpuTimeExceedLimit
	ClockTimeExceedLimit
	MemoryExceedLimit
	ThreadExceedLimit
	OutputSizeExceedLimit
	BadSyscall
	NormalRuntimeError
	SandboxError
)

var ErrorString = []string{
	"OK",
	"cpu time exceed limit",
	"clock time exceed limit",
	"memory exceed limit",
	"thread exceed limit",
	"output size exceed limit",
	"bad syscall",
	"normal runtime error",
	"sandbox error",
}

type Error struct {
	ErrorNum int
	Helper   string
}

func (e *Error) Error() string {
	if len(e.Helper) != 0 {
		return "sandbox: " + ErrorString[e.ErrorNum] + ", " + e.Helper
	} else {
		return "sandbox: " + ErrorString[e.ErrorNum]
	}
}
