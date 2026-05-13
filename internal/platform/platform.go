package platform

import (
	"fmt"
	"os"
	"os/exec"
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

// Check verifies the runtime environment is suitable for wsl-obsidian-clip.
func Check() error {
	if !isWSL() {
		return fmt.Errorf("not running under WSL")
	}
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		return fmt.Errorf("powershell.exe not found in PATH")
	}
	if os.Getenv("DISPLAY") == "" {
		return fmt.Errorf("DISPLAY is not set; start this from a WSLg graphical session with X11 clipboard access")
	}
	if err := canConnectX11(); err != nil {
		return fmt.Errorf("X11 clipboard is not reachable on DISPLAY=%q: %w", os.Getenv("DISPLAY"), err)
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
