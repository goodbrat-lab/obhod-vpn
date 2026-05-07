package logger

import (
	"fmt"
	"log"
	"log/syslog"
)

var sysLog *syslog.Writer

func Init() error {
	var err error
	sysLog, err = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "obhoud")
	if err != nil {
		return err
	}
	log.SetOutput(sysLog)
	return nil
}

func Info(format string, v ...interface{}) {
	if sysLog != nil {
		sysLog.Info(fmt.Sprintf(format, v...))
	} else {
		log.Printf("[INFO] "+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if sysLog != nil {
		sysLog.Err(fmt.Sprintf(format, v...))
	} else {
		log.Printf("[ERROR] "+format, v...)
	}
}
