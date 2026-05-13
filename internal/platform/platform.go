package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Check verifies the runtime environment is suitable for wsl-obsidian-clip.
func Check() error {
	if !isWSL() {
		return fmt.Errorf("not running under WSL")
	}
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		return fmt.Errorf("powershell.exe not found in PATH")
	}
	return nil
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}
