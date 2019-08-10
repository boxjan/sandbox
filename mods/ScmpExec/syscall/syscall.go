// +build linux

package scmpSyscall

import (
	"syscall"
	_ "syscall"
	_ "unsafe"
)

//go:linkname beforeFork syscall.runtime_BeforeFork
func beforeFork()

//go:linkname afterFork syscall.runtime_AfterFork
func afterFork()

//go:linkname afterForkInChild syscall.runtime_AfterForkInChild
func afterForkInChild()

//go:linkname rawSyscallNoError syscall.rawSyscallNoError
func rawSyscallNoError(trap, a1, a2, a3 uintptr) (r1, r2 uintptr)

//go:linkname rawVforkSyscall syscall.rawVforkSyscall
func rawVforkSyscall(trap, a1 uintptr) (r1 uintptr, err syscall.Errno)

//go:linkname fcntl syscall.fcntl
func fcntl(fd int, cmd int, arg int) (val int, err error)

//go:linkname readlen syscall.readlen
func readlen(fd int, p *byte, np int) (n int, err error)

func itoa(val int) string { // do it here rather than with fmt to avoid dependency
	if val < 0 {
		return "-" + uitoa(uint(-val))
	}
	return uitoa(uint(val))
}

func uitoa(val uint) string {
	var buf [32]byte // big enough for int64
	i := len(buf) - 1
	for val >= 10 {
		buf[i] = byte(val%10 + '0')
		i--
		val /= 10
	}
	buf[i] = byte(val + '0')
	return string(buf[i:])
}

// writeIDMappings writes the user namespace User ID or Group ID mappings to the specified path.
func writeIDMappings(path string, idMap []SysProcIDMap) error {
	fd, err := syscall.Open(path, syscall.O_RDWR, 0)
	if err != nil {
		return err
	}

	data := ""
	for _, im := range idMap {
		data = data + itoa(im.ContainerID) + " " + itoa(im.HostID) + " " + itoa(im.Size) + "\n"
	}

	bytes, err := syscall.ByteSliceFromString(data)
	if err != nil {
		_ = syscall.Close(fd)
		return err
	}

	if _, err := syscall.Write(fd, bytes); err != nil {
		_ = syscall.Close(fd)
		return err
	}

	if err := syscall.Close(fd); err != nil {
		return err
	}

	return nil
}

// writeSetgroups writes to /proc/PID/setgroups "deny" if enable is false
// and "allow" if enable is true.
// This is needed since kernel 3.19, because you can't write gid_map without
// disabling setgroups() system call.
func writeSetgroups(pid int, enable bool) error {
	sgf := "/proc/" + itoa(pid) + "/setgroups"
	fd, err := syscall.Open(sgf, syscall.O_RDWR, 0)
	if err != nil {
		return err
	}

	var data []byte
	if enable {
		data = []byte("allow")
	} else {
		data = []byte("deny")
	}

	if _, err := syscall.Write(fd, data); err != nil {
		_ = syscall.Close(fd)
		return err
	}

	return syscall.Close(fd)
}

// writeUidGidMappings writes User ID and Group ID mappings for user namespaces
// for a process and it is called from the parent process.
func writeUidGidMappings(pid int, sys *SysProcAttr) error {
	if sys.UidMappings != nil {
		uidf := "/proc/" + itoa(pid) + "/uid_map"
		if err := writeIDMappings(uidf, sys.UidMappings); err != nil {
			return err
		}
	}

	if sys.GidMappings != nil {
		// If the kernel is too old to support /proc/PID/setgroups, writeSetGroups will return ENOENT; this is OK.
		if err := writeSetgroups(pid, sys.GidMappingsEnableSetgroups); err != nil && err != syscall.ENOENT {
			return err
		}
		gidf := "/proc/" + itoa(pid) + "/gid_map"
		if err := writeIDMappings(gidf, sys.GidMappings); err != nil {
			return err
		}
	}

	return nil
}
