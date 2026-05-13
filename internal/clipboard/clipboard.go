package clipboard

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
)

//go:embed clipboard.ps1
var psScript string

// Client manages a persistent PowerShell process for clipboard operations.
type Client struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
	mu      sync.Mutex
	logger  *log.Logger
	verbose bool
}

// newPSCommand creates the exec.Cmd for the PowerShell subprocess.
var newPSCommand = func() *exec.Cmd {
	return exec.Command("powershell.exe",
		"-STA", "-NoLogo", "-NoProfile", "-NonInteractive",
		"-Command", psScript,
	)
}

// NewClient spawns a persistent powershell.exe -STA process and waits for READY.
func NewClient(logger *log.Logger, verbose bool) (*Client, error) {
	cmd := newPSCommand()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start powershell: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 32*1024*1024)

	if !scanner.Scan() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("waiting for READY: %w", err)
		}
		return nil, fmt.Errorf("powershell exited before READY")
	}
	if line := strings.TrimSpace(scanner.Text()); line != "READY" {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("expected READY, got %q", line)
	}

	logger.Println("PowerShell clipboard client started")

	return &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  scanner,
		logger:  logger,
		verbose: verbose,
	}, nil
}

// Check queries the clipboard for an image. Returns PNG bytes if an image
// is present, or nil if the clipboard is empty or contains non-image data.
func (c *Client) Check() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.verbose {
		c.logger.Println("[ps:send] CHECK")
	}
	if _, err := fmt.Fprintln(c.stdin, "CHECK"); err != nil {
		return nil, fmt.Errorf("send CHECK: %w", err)
	}

	if !c.stdout.Scan() {
		if err := c.stdout.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("powershell process exited")
	}

	line := strings.TrimSpace(c.stdout.Text())
	if c.verbose {
		c.logger.Printf("[ps:recv] %s", line)
	}

	switch line {
	case "NONE":
		return nil, nil
	case "IMAGE":
		if !c.stdout.Scan() {
			return nil, fmt.Errorf("read base64: powershell process exited")
		}
		b64 := strings.TrimSpace(c.stdout.Text())
		if c.verbose {
			c.logger.Printf("[ps:recv] IMAGE data (%d chars base64)", len(b64))
		}

		// Read END marker
		if !c.stdout.Scan() {
			return nil, fmt.Errorf("read END marker: powershell process exited")
		}
		if end := strings.TrimSpace(c.stdout.Text()); end != "END" {
			return nil, fmt.Errorf("expected END, got %q", end)
		}
		if c.verbose {
			c.logger.Println("[ps:recv] END")
		}

		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("decode base64: %w", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unexpected response: %q", line)
	}
}

// Close gracefully shuts down the PowerShell process.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, _ = fmt.Fprintln(c.stdin, "EXIT")
	_ = c.stdin.Close()
	return c.cmd.Wait()
}
