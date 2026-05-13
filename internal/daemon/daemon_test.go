package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStatusRemovesStalePidFile(t *testing.T) {
	t.Setenv(daemonChildEnv, "")
	oldPidFile := PidFile
	PidFile = filepath.Join(t.TempDir(), "wsl-obsidian-clip.pid")
	t.Cleanup(func() { PidFile = oldPidFile })

	if err := os.WriteFile(PidFile, []byte("999999"), 0644); err != nil {
		t.Fatalf("write stale pid file: %v", err)
	}

	_, err := Status()
	if err == nil || !strings.Contains(err.Error(), "not running") {
		t.Fatalf("expected not running error, got %v", err)
	}
	if _, statErr := os.Stat(PidFile); !os.IsNotExist(statErr) {
		t.Fatalf("expected stale pid file to be removed, stat err=%v", statErr)
	}
}

func TestMarkChildRunningWritesAndCleansPidFile(t *testing.T) {
	t.Setenv(daemonChildEnv, "1")
	oldPidFile := PidFile
	PidFile = filepath.Join(t.TempDir(), "wsl-obsidian-clip.pid")
	t.Cleanup(func() { PidFile = oldPidFile })

	cleanup, err := MarkChildRunning()
	if err != nil {
		t.Fatalf("MarkChildRunning: %v", err)
	}
	if cleanup == nil {
		t.Fatal("expected cleanup function")
	}

	data, err := os.ReadFile(PidFile)
	if err != nil {
		t.Fatalf("read pid file: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != strconv.Itoa(os.Getpid()) {
		t.Fatalf("expected pid file to contain current pid, got %q", got)
	}

	cleanup()
	if _, err := os.Stat(PidFile); !os.IsNotExist(err) {
		t.Fatalf("expected pid file to be removed, stat err=%v", err)
	}
}

func TestStopTerminatesProcessAndRemovesPidFile(t *testing.T) {
	t.Setenv(daemonChildEnv, "")
	oldPidFile := PidFile
	PidFile = filepath.Join(t.TempDir(), "wsl-obsidian-clip.pid")
	t.Cleanup(func() { PidFile = oldPidFile })

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-waitDone:
		case <-time.After(100 * time.Millisecond):
		}
	})

	if err := os.WriteFile(PidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	if err := Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("process did not exit")
	}

	if _, err := os.Stat(PidFile); !os.IsNotExist(err) {
		t.Fatalf("expected pid file to be removed, stat err=%v", err)
	}
}

func TestDaemonizeQuietTreatsAlreadyRunningAsSuccess(t *testing.T) {
	t.Setenv(daemonChildEnv, "")
	oldPidFile := PidFile
	PidFile = filepath.Join(t.TempDir(), "wsl-obsidian-clip.pid")
	t.Cleanup(func() { PidFile = oldPidFile })

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-waitDone:
		case <-time.After(100 * time.Millisecond):
		}
	})

	if err := os.WriteFile(PidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	if err := Daemonize(true); err != nil {
		t.Fatalf("Daemonize quiet should ignore running daemon: %v", err)
	}

	err := Daemonize(false)
	if err == nil || !strings.Contains(err.Error(), "daemon already running") {
		t.Fatalf("expected already running error, got %v", err)
	}
}
