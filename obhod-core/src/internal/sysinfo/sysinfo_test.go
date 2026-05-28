package sysinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"1.1.1.1", true},
		{"8.8.8.8", true},
		{"256.0.0.1", false},
		{"invalid-ip", false},
		{"2001:db8::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := isValidIP(tt.ip); got != tt.want {
				t.Errorf("isValidIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestSafeReadFile(t *testing.T) {
	// Create a temporary file in the allowed prefix /tmp/
	tmpFile, err := os.CreateTemp("", "safe_read_test_")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte("test content")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// 1. Reading from allowed /tmp prefix
	data, err := safeReadFile(tmpFile.Name())
	if err != nil {
		t.Errorf("safeReadFile() unexpected error for allowed path: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("safeReadFile() got %s, want %s", string(data), string(content))
	}

	// 2. Reading from disallowed directory (e.g. windows style or arbitrary root if non-unix)
	// We want to test path traversal prevention
	traversalPath := filepath.Join(tmpFile.Name(), "..", "..", "disallowed.txt")
	_, err = safeReadFile(traversalPath)
	if err == nil {
		t.Error("safeReadFile() expected error for path traversal / disallowed path, got nil")
	}
}
