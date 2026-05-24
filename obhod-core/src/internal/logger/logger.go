package logger

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var (
	currentLevel Level = LevelInfo
)

func Init(levelStr string) error {
	currentLevel = parseLevel(levelStr)
	return initSyslog()
}

func parseLevel(l string) Level {
	switch strings.ToLower(l) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

func logMessage(level Level, component, context, format string, v ...interface{}) {
	if level < currentLevel {
		return
	}

	levelStr := "INFO"
	switch level {
	case LevelDebug:
		levelStr = "DEBUG"
	case LevelInfo:
		levelStr = "INFO"
	case LevelWarn:
		levelStr = "WARN"
	case LevelError:
		levelStr = "ERROR"
	case LevelFatal:
		levelStr = "FATAL"
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	formatted := fmt.Sprintf("[%s] [%s] [%s] [%s] %s", timestamp, levelStr, component, context, msg)

	if hasSyslog() {
		writeToSyslog(level, formatted)
	} else {
		fmt.Fprintln(os.Stderr, formatted)
	}

	if level == LevelFatal {
		os.Exit(1)
	}
}

func Debug(component, context, format string, v ...interface{}) {
	logMessage(LevelDebug, component, context, format, v...)
}

func Info(component, context, format string, v ...interface{}) {
	logMessage(LevelInfo, component, context, format, v...)
}

func Warn(component, context, format string, v ...interface{}) {
	logMessage(LevelWarn, component, context, format, v...)
}

func Error(component, context, format string, v ...interface{}) {
	logMessage(LevelError, component, context, format, v...)
}

func Fatal(component, context, format string, v ...interface{}) {
	logMessage(LevelFatal, component, context, format, v...)
}
