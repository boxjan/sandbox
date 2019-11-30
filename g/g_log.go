package g

import (
	"github.com/boxjan/golib/logs"
	"github.com/sdibtacm/sandbox/exec/log"
)

func GetLog() *logs.Logger {
	return log.GetLog()
}

func SetLog(logger *logs.Logger) {
	log.SetLog(logger)
}

func CloseLog() {
	log.CloseLog()
}
