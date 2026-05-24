//go:build linux

package logger

import (
	"log"
	"log/syslog"
)

var sysLog *syslog.Writer

func initSyslog() error {
	var err error
	sysLog, err = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "obhod")
	if err != nil {
		return err
	}
	log.SetOutput(sysLog)
	return nil
}

func hasSyslog() bool {
	return sysLog != nil
}

func writeToSyslog(level Level, formatted string) {
	if sysLog != nil {
		switch level {
		case LevelDebug:
			sysLog.Debug(formatted)
		case LevelInfo:
			sysLog.Info(formatted)
		case LevelWarn:
			sysLog.Warning(formatted)
		case LevelError:
			sysLog.Err(formatted)
		case LevelFatal:
			sysLog.Crit(formatted)
		}
	}
}
