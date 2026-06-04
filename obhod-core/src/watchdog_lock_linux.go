//go:build !windows

package main

import (
	"os"
	"syscall"
)

func lockInstance(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}
