package exec

const (
	RLIMIT_UNRESOURCE    uint64 = 0
	TIME_UNRESOURCE      uint   = 0
	BYTE_UNRESOURCE      uint64 = RLIMIT_UNRESOURCE
	MAX_TIME             uint   = 0xFFFFFFFF
	SUGGEST_THREAD_LIMIT uint   = 1024
	MAX_THREAD_LIMIT     uint   = 65535
	MIN_MEMORY_LIMIT     uint64 = 16777216 // byte 16M
	RLIMIT_STACK_MIN     uint64 = 8388608  // byte 8M
	SECCOMP
)

const (
	SANDBOX_NO_START = iota
	SANDBOX_PREPARE_PIPE
	SANDBOX_READY_FOR_CLONE
	SANDBOX_READY_FOR_CHROOT
	SANDBOX_READY_FOR_SETUID
	SANDBOX_READY_FOR_SETGID
	SANDBOX_READY_FOR_SETUMASK
	SANDBOX_READY_FOR_CHDIR
	SANDBOX_READY_FOR_SET_PDEATHSIG
	SANDBOX_READY_FOR_PDEATHSIG_KILL_MYSELF
	SANDBOX_READY_FRO_DUP_FILE
	SANDBOX_READY_FOR_SET_RLIMIT
	SANDBOX_READY_FOR_SET_PTRACE
	SANDBOX_READY_FOR_SET_BPF
	SANDBOX_READY_FOR_EXEC

	SANDBOX_READ_PIPE
)

var SANDBOX_STEP_STR = []string{
	"no start",
	"prepare pipe",
	"clone",
	"chroot",
	"set uid",
	"set gid",
	"set umask",
	"chdir",
	"set pdeathsig",
	"parent died, kill myself",
	"dup files",
	"set rlimit",
	"set ptrace",
	"set bpf",
	"exec",
	"read error status from pipe",
}
