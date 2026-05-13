package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckRequiresDisplay(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("DISPLAY", "")
	t.Setenv("PATH", binDir)

	oldReadProcVersion := readProcVersion
	oldCanConnect := canConnectX11
	t.Cleanup(func() {
		readProcVersion = oldReadProcVersion
		canConnectX11 = oldCanConnect
	})

	readProcVersion = func() ([]byte, error) {
		return []byte("Linux version microsoft"), nil
	}
	canConnectX11 = func() error {
		t.Fatal("canConnectX11 should not be called when DISPLAY is missing")
		return nil
	}

	if err := writeExecutable(binDir, "powershell.exe"); err != nil {
		t.Fatalf("write powershell.exe: %v", err)
	}

	err := Check()
	if err == nil || !strings.Contains(err.Error(), "DISPLAY is not set") {
		t.Fatalf("expected DISPLAY error, got %v", err)
	}
}

func TestCheckReportsX11ConnectionFailure(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	t.Setenv("DISPLAY", ":0")

	oldReadProcVersion := readProcVersion
	oldCanConnect := canConnectX11
	t.Cleanup(func() {
		readProcVersion = oldReadProcVersion
		canConnectX11 = oldCanConnect
	})

	readProcVersion = func() ([]byte, error) {
		return []byte("Linux version microsoft"), nil
	}
	canConnectX11 = func() error {
		return errors.New("dial tcp: connection refused")
	}

	if err := writeExecutable(binDir, "powershell.exe"); err != nil {
		t.Fatalf("write powershell.exe: %v", err)
	}

	err := Check()
	if err == nil || !strings.Contains(err.Error(), "X11 clipboard is not reachable") {
		t.Fatalf("expected X11 connection error, got %v", err)
	}
}

func writeExecutable(dir string, name string) error {
	path := filepath.Join(dir, name)
	return os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755)
}
