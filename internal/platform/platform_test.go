package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckRequiresDisplay(t *testing.T) {
	t.Setenv("DISPLAY", "")

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

	err := Check()
	if err == nil || !strings.Contains(err.Error(), "DISPLAY is not set") {
		t.Fatalf("expected DISPLAY error, got %v", err)
	}
}

func TestCheckReportsX11ConnectionFailure(t *testing.T) {
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

	err := Check()
	if err == nil || !strings.Contains(err.Error(), "X11 clipboard is not reachable") {
		t.Fatalf("expected X11 connection error, got %v", err)
	}
}

func TestResolvePowerShellUsesOverride(t *testing.T) {
	binDir := t.TempDir()
	path := filepath.Join(binDir, "custom-powershell.exe")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write override: %v", err)
	}

	got, err := ResolvePowerShell(path)
	if err != nil {
		t.Fatalf("ResolvePowerShell: %v", err)
	}
	if got != path {
		t.Fatalf("ResolvePowerShell() = %q, want %q", got, path)
	}
}

func TestResolvePowerShellUsesKnownWindowsPath(t *testing.T) {
	binDir := t.TempDir()
	path := filepath.Join(binDir, "powershell.exe")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write fallback: %v", err)
	}

	oldLookPath := lookPath
	oldWindowsPowerShellPaths := windowsPowerShellPaths
	t.Cleanup(func() {
		lookPath = oldLookPath
		windowsPowerShellPaths = oldWindowsPowerShellPaths
	})
	lookPath = func(string) (string, error) {
		return "", os.ErrNotExist
	}
	windowsPowerShellPaths = []string{path}

	got, err := ResolvePowerShell("")
	if err != nil {
		t.Fatalf("ResolvePowerShell: %v", err)
	}
	if got != path {
		t.Fatalf("ResolvePowerShell() = %q, want %q", got, path)
	}
}

func writeExecutable(dir string, name string) error {
	path := filepath.Join(dir, name)
	return os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755)
}
