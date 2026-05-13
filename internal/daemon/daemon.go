package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	PidFile = "/tmp/.wsl-obsidian-clip.pid"
	LogFile = "/tmp/.wsl-obsidian-clip.log"
)

const daemonChildEnv = "WSL_SNIP_DAEMON_CHILD"

// Daemonize re-execs the current process as a detached background daemon.
func Daemonize(quiet bool) error {
	if pid, err := runningPID(); err == nil {
		if quiet {
			return nil
		}
		return fmt.Errorf("daemon already running (PID %d)", pid)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	// Rebuild args without --daemon / -d
	var args []string
	for _, a := range os.Args[1:] {
		if a == "--daemon" || a == "-d" {
			continue
		}
		args = append(args, a)
	}

	logF, err := os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = logF
	cmd.Stderr = logF
	cmd.Env = append(os.Environ(), daemonChildEnv+"=1")

	if err := cmd.Start(); err != nil {
		logF.Close()
		return fmt.Errorf("start daemon: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	pid := cmd.Process.Pid
	if err := waitForStartup(pid, waitCh); err != nil {
		logF.Close()
		return err
	}

	if !quiet {
		fmt.Printf("wsl-obsidian-clip daemon started (PID %d)\n", pid)
	}
	logF.Close()
	return nil
}

// Stop sends SIGTERM to the running daemon.
func Stop() error {
	pid, err := runningPID()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("not running")
		}
		return err
	}

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			_ = os.Remove(PidFile)
			return fmt.Errorf("not running")
		}
		return fmt.Errorf("signal process %d: %w", pid, err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		alive, err := processExists(pid)
		if err != nil {
			return fmt.Errorf("check process %d: %w", pid, err)
		}
		if !alive {
			_ = os.Remove(PidFile)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	if alive, err := processExists(pid); err != nil {
		return fmt.Errorf("check process %d: %w", pid, err)
	} else if alive {
		return fmt.Errorf("process %d did not exit after SIGTERM", pid)
	}

	_ = os.Remove(PidFile)
	return nil
}

// Status returns information about the running daemon.
func Status() (string, error) {
	pid, err := runningPID()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("not running")
		}
		return "", err
	}

	return fmt.Sprintf("wsl-obsidian-clip daemon is running (PID %d)", pid), nil
}

// MarkChildRunning writes the PID file for the background child process and
// returns a cleanup function that removes it on exit.
func MarkChildRunning() (func(), error) {
	if os.Getenv(daemonChildEnv) == "" {
		return nil, nil
	}

	pid := os.Getpid()
	if existingPID, err := runningPID(); err == nil && existingPID != pid {
		return nil, fmt.Errorf("daemon already running (PID %d)", existingPID)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if err := os.WriteFile(PidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return nil, fmt.Errorf("write PID file: %w", err)
	}

	return func() {
		_ = os.Remove(PidFile)
	}, nil
}

func readPid() (int, error) {
	data, err := os.ReadFile(PidFile)
	if err != nil {
		return 0, fmt.Errorf("read PID file: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse PID: %w", err)
	}
	return pid, nil
}

func waitForStartup(pid int, waitCh <-chan error) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-waitCh:
			if err != nil {
				return fmt.Errorf("daemon exited during startup: %w", err)
			}
			return fmt.Errorf("daemon exited during startup")
		default:
		}

		runningPID, err := runningPID()
		if err == nil && runningPID == pid {
			return nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		time.Sleep(50 * time.Millisecond)
	}

	select {
	case err := <-waitCh:
		if err != nil {
			return fmt.Errorf("daemon exited during startup: %w", err)
		}
		return fmt.Errorf("daemon exited during startup")
	default:
		return fmt.Errorf("daemon did not report startup")
	}
}

func runningPID() (int, error) {
	pid, err := readPid()
	if err != nil {
		if os.IsNotExist(err) {
			return 0, os.ErrNotExist
		}
		return 0, err
	}

	alive, err := processExists(pid)
	if err != nil {
		return 0, fmt.Errorf("check process %d: %w", pid, err)
	}
	if !alive {
		_ = os.Remove(PidFile)
		return 0, os.ErrNotExist
	}
	return pid, nil
}

func processExists(pid int) (bool, error) {
	err := syscall.Kill(pid, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true, nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	return false, err
}
