package scmpFilter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/sdibtacm/sandbox/exec/log"
	"github.com/sdibtacm/sandbox/g"
	"github.com/sdibtacm/sandbox/units/helper"
	"github.com/sdibtacm/sandbox/units/seccomp"
	"io"
	"io/ioutil"
	"os"
	"syscall"
	"unsafe"
)

const (
	EPERM  = int16(syscall.EPERM)
	ENOSYS = int16(syscall.ENOSYS)
)

const (
	DEFAULT_KILL   ScmpAction = 0x00
	DEFAULT_TRACE  ScmpAction = 0x01
	DEFAULT_EPERM  ScmpAction = 0x02
	DEFAULT_ENOSYS ScmpAction = 0x03

	OTHERS_KILL   ScmpAction = 0x10
	OTHERS_TRACE  ScmpAction = 0x11
	OTHERS_EPERM  ScmpAction = 0x12
	OTHERS_ENOSYS ScmpAction = 0x13
)

var (
	ErrScmpNotAllowDefaultActionAllow = errors.New("only lrun filter can let default action is 'act_allow' ")
)

type scmpMid struct {
	seccomp.ScmpFilter
}

type ScmpAction int8

type ScmpFilterLoadHelper struct {
	Action ScmpAction

	LrunScmpFilter string // https://github.com/quark-zju/lrun/blob/master/src/seccomp.h

	Level int

	ExecvePathPointer unsafe.Pointer
}

type scmpLoadFilter struct {
	SetPrivs bool
	BPF      *syscall.SockFprog
}

func GetScmpFilter(helper *ScmpFilterLoadHelper) (filter *scmpLoadFilter, err error) {
	if helper.Level == 0 {
		return nil, nil
	}

	var scmp *seccomp.ScmpFilter
	defer func() {
		if scmp != nil {
			scmp.Release()
		}
		// we hope will release after function return
	}()

	if helper.Level == -1 {
		// will load by lrun scmp filter string
		scmp, err = lrunFilterParse(helper)
	} else {
		// load as white list
		scmp, err = nFilterParse(helper)
	}
	if err != nil {
		return
	}

	midFilter := scmpLoadFilter{}

	midFilter.SetPrivs = true
	midFilter.BPF, err = scmpToBPF(scmp)
	if err != nil {
		return
	}

	filter = &midFilter
	return
}

func nFilterParse(helper *ScmpFilterLoadHelper) (scmp *seccomp.ScmpFilter, err error) {
	if helper.Action&0x10 == 0x10 {
		err = ErrScmpNotAllowDefaultActionAllow
		g.GetLog().Warning("{}", err)
		return
	}

	switch helper.Action | 0x00 {
	case 0x00:
		scmp, err = seccomp.NewFilter(seccomp.ActKill)
	case 0x01:
		scmp, err = seccomp.NewFilter(seccomp.ActTrace)
	case 0x02:
		scmp, err = seccomp.NewFilter(seccomp.ActErrno.SetReturnCode(EPERM))
	case 0x03:
		scmp, err = seccomp.NewFilter(seccomp.ActErrno.SetReturnCode(ENOSYS))
	}

	switch helper.Level {
	case 1:

	}

	return
}

func scmpToBPF(filter *seccomp.ScmpFilter) (BPF *syscall.SockFprog, err error) {

	scmpBPFTempFile, err := ioutil.TempFile("", "sandbox-ScmpBPF-")
	defer func() {
		if scmpBPFTempFile != nil {
			_ = scmpBPFTempFile.Close()
			os.Remove(scmpBPFTempFile.Name())
		}
	}()

	if err != nil {
		g.GetLog().Warning("can not make bpf temp file with error: {}", err)
		return
	}

	err = filter.ExportBPF2Fd(scmpBPFTempFile.Fd())
	if err != nil {
		return
	}
	_ = scmpBPFTempFile.Sync()

	BpfFileStat, _ := scmpBPFTempFile.Stat()
	sockFilterFileSize := BpfFileStat.Size()
	if sockFilterFileSize%8 != 0 {
		err = errors.New("sockFilterFileSize error " + string(sockFilterFileSize))
		return
	}
	sockFilters := make([]syscall.SockFilter, sockFilterFileSize/8)
	var ret = sockFilterFileSize
	ret, err = scmpBPFTempFile.Seek(0, io.SeekStart)
	if err != nil {
		return
	}
	if ret != 0 {
		log.GetLog().Warning("seek fail, ret not at file start, now at:{}", ret)
		err = errors.New("seek fail, ret not at file start")
	}

	sockFilterFileContent, err := ioutil.ReadAll(scmpBPFTempFile)
	bytesBuffer := bytes.NewBuffer(sockFilterFileContent)
	if helper.IsLittleEndian() {
		err = binary.Read(bytesBuffer, binary.LittleEndian, &sockFilters)
	} else {
		err = binary.Read(bytesBuffer, binary.BigEndian, &sockFilters)
	}

	BPF = &syscall.SockFprog{
		Len:    uint16(sockFilterFileSize / 8),
		Filter: &sockFilters[0],
	}
	return
}
