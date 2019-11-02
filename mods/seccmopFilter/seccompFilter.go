// +build linux

package seccmopFilter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/sdibtacm/sandbox/mods/seccomp"
	"io/ioutil"
	"os"
	"syscall"
	"unsafe"
)

type scmpF struct {
	scmp *seccomp.ScmpFilter
}

func GetFilterByLevel(level int8, usePtrace bool, execPathPtr unsafe.Pointer,
) (bpf *syscall.SockFprog, setNoPrive bool, err error) {

	if level == 0 {
		return
	}

	var scmp *seccomp.ScmpFilter
	if usePtrace {
		scmp, err = seccomp.NewFilter(seccomp.ActTrace.SetReturnCode(0))
	} else {
		scmp, err = seccomp.NewFilter(seccomp.ActKill)
	}
	if err != nil {
		return
	}

	s := &scmpF{scmp: scmp}

	err = s.allowExec(execPathPtr)
	if err != nil {
		return
	}

	scmpBpfFile, err := ioutil.TempFile("", "scmpBpfTempFile-")
	defer os.Remove(scmpBpfFile.Name())
	defer scmpBpfFile.Close()
	scmpPfcFile, err := os.OpenFile("scmpBpfFile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

	if err != nil {
		panic(err)
	}
	_ = scmp.ExportPFC(scmpPfcFile)
	scmpPfcFile.Close()
	err = scmp.ExportBPF(scmpBpfFile)
	if err != nil {
		return
	}
	_ = scmpBpfFile.Sync()
	scmp.Release()
	scmpBpfFileStat, _ := scmpBpfFile.Stat()
	sockFilterFileSize := scmpBpfFileStat.Size()

	if sockFilterFileSize%8 != 0 {
		err = errors.New("sockFilterFileSize error " + string(sockFilterFileSize))
		return
	}

	sockFilters := make([]syscall.SockFilter, sockFilterFileSize/8)
	_, err = scmpBpfFile.Seek(0, os.SEEK_SET)
	if err != nil {
		return
	}

	sockFilterFileContent, err := ioutil.ReadAll(scmpBpfFile)
	bytesBuffer := bytes.NewBuffer(sockFilterFileContent)
	err = binary.Read(bytesBuffer, binary.LittleEndian, &sockFilters)

	bpf = &syscall.SockFprog{
		Len:    uint16(sockFilterFileSize / 8),
		Filter: &sockFilters[0],
	}
	setNoPrive = true
	return
}

func (s *scmpF) allowExec(pointer unsafe.Pointer) (err error) {

	call, err := seccomp.GetSyscallFromName("execve")
	if err != nil {
		return
	}

	cond, err := seccomp.MakeCondition(0, seccomp.CompareEqual, uint64(uintptr(pointer)))
	if err != nil {
		return
	}
	conds := []seccomp.ScmpCondition{cond}

	err = s.scmp.AddRuleConditional(call, seccomp.ActAllow, conds)
	if err != nil {
		return
	}
	return
}
