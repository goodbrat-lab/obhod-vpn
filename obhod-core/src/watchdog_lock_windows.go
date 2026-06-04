//go:build windows

package main

import (
	"os"
)

func lockInstance(file *os.File) error {
	// Stub for Windows compilation
	return nil
}
