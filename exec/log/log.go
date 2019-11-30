package log

import "github.com/boxjan/golib/logs"

var log *logs.Logger

func GetLog() *logs.Logger {
	if log == nil {
		log = logs.NewLoggerWithCmdWriterWithTraceLevel()
		log.Warning("You did't not set before, so new one")
	}
	return log
}

func SetLog(logger *logs.Logger) {
	if log != nil {
		log.Warning("logger will be change")
	}
	log = logger
}

func CloseLog() {
	if log != nil {
		log.Warning("will stop use this logger now")
		log = nil
	}
}
