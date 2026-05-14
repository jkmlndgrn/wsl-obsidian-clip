package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/xgb"
)

var readProcVersion = func() ([]byte, error) {
	return os.ReadFile("/proc/version")
}

var canConnectX11 = func() error {
	conn, err := xgb.NewConn()
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

var lookPath = exec.LookPath

var windowsPowerShellPaths = []string{
	"/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe",
}

// Check verifies the runtime environment is suitable for wsl-obsidian-clip.
func Check() error {
	if !isWSL() {
		return fmt.Errorf("not running under WSL")
	}
	if os.Getenv("DISPLAY") == "" {
		return fmt.Errorf("DISPLAY is not set; start this from a WSLg graphical session with X11 clipboard access")
	}
	if err := canConnectX11(); err != nil {
		return fmt.Errorf("X11 clipboard is not reachable on DISPLAY=%q: %w", os.Getenv("DISPLAY"), err)
	}
	return nil
}

func ResolvePowerShell(overridePath string) (string, error) {
	if overridePath != "" {
		if err := executableExists(overridePath); err != nil {
			return "", fmt.Errorf("powershell_path_override is not executable: %s: %w", overridePath, err)
		}
		return overridePath, nil
	}

	if path, err := lookPath("powershell.exe"); err == nil {
		return path, nil
	}
	for _, path := range windowsPowerShellPaths {
		if err := executableExists(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("powershell.exe not found in PATH or known WSL Windows paths")
}

func executableExists(path string) error {
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("path is a directory")
	}
	if info.Mode()&0111 == 0 {
		return errors.New("missing executable bit")
	}
	return nil
}

func isWSL() bool {
	data, err := readProcVersion()
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}
