package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadRawInputFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "message.eml")
	want := "From: sender@example.com\r\nSubject: test\r\n\r\nhello\r\n"
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	raw, err := readRawInput([]string{"-F", path})
	if err != nil {
		t.Fatalf("readRawInput() error = %v", err)
	}
	if got := string(raw); got != want {
		t.Fatalf("readRawInput() = %q, want %q", got, want)
	}
}
