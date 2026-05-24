//go:build !linux

package logger

func initSyslog() error {
	// Syslog is not supported on non-Linux platforms, fallback to console (stderr)
	return nil
}

func hasSyslog() bool {
	return false
}

func writeToSyslog(level Level, formatted string) {
	// No-op
}
