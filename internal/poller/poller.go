package poller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxConsecutiveErrors = 5

// Clipboard is the interface for clipboard access.
type Clipboard interface {
	Check() ([]byte, error)
	Close() error
}

// EmbedWriter is the interface for lazily offering embeds to the clipboard.
type EmbedWriter interface {
	OfferEmbed(filename string, onPaste func() error) error
	Close() error
}

// ClientFactory creates a new clipboard client.
type ClientFactory func() (Clipboard, error)

// Run polls the clipboard at the given interval until the context is cancelled.
func Run(ctx context.Context, logger *log.Logger, interval int, outputDir string, newClient ClientFactory, embedWriter EmbedWriter) error {
	client, err := newClient()
	if err != nil {
		return fmt.Errorf("start clipboard client: %w", err)
	}
	defer func() { _ = client.Close() }()
	defer func() { _ = embedWriter.Close() }()

	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()

	state := &pollState{
		knownHashes: make(map[string]bool),
		outputDir:   outputDir,
	}
	// Index existing files in attachment dir
	state.indexExisting(logger)

	consecutiveErrors := 0

	logger.Printf("Polling started (interval: %dms)", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Println("Shutting down...")
			return nil
		case <-ticker.C:
			if err := poll(client, logger, state, embedWriter); err != nil {
				consecutiveErrors++
				logger.Printf("Poll error (%d/%d): %v", consecutiveErrors, maxConsecutiveErrors, err)

				if consecutiveErrors >= maxConsecutiveErrors {
					logger.Println("Too many errors, restarting PowerShell client...")
					_ = client.Close()

					client, err = newClient()
					if err != nil {
						return fmt.Errorf("restart clipboard client: %w", err)
					}
					consecutiveErrors = 0
				}
			} else {
				consecutiveErrors = 0
			}
		}
	}
}

type pollState struct {
	mu          sync.Mutex
	knownHashes map[string]bool
	outputDir   string
	lastHash    string // track the last clipboard image hash to avoid re-processing
}

// indexExisting scans the output directory and records hashes of existing PNG files.
func (s *pollState) indexExisting(logger *log.Logger) {
	entries, err := os.ReadDir(s.outputDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".png" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.outputDir, e.Name()))
		if err != nil {
			continue
		}
		hash := hashBytes(data)
		s.knownHashes[hash] = true
	}
	if len(s.knownHashes) > 0 {
		logger.Printf("Indexed %d existing screenshots", len(s.knownHashes))
	}
}

func poll(client Clipboard, logger *log.Logger, state *pollState, embedWriter EmbedWriter) error {
	pngData, err := client.Check()
	if err != nil {
		return fmt.Errorf("check clipboard: %w", err)
	}
	if pngData == nil {
		state.resetLastHash()
		return nil
	}

	hash := hashBytes(pngData)
	if !state.shouldOffer(hash) {
		return nil
	}

	// Generate timestamp filename
	filename := generateFilename(state.outputDir)
	if err := embedWriter.OfferEmbed(filename, func() error {
		return state.savePending(logger, filename, pngData, hash)
	}); err != nil {
		return fmt.Errorf("offer embed %s: %w", filename, err)
	}

	logger.Printf("Clipboard ready: ![[%s]]", filename)
	return nil
}

func (s *pollState) resetLastHash() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastHash = ""
}

func (s *pollState) shouldOffer(hash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if hash == s.lastHash {
		return false
	}
	s.lastHash = hash

	return !s.knownHashes[hash]
}

func (s *pollState) savePending(logger *log.Logger, filename string, pngData []byte, hash string) error {
	filePath := filepath.Join(s.outputDir, filename)

	s.mu.Lock()
	if s.knownHashes[hash] {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if err := os.WriteFile(filePath, pngData, 0644); err != nil {
		return fmt.Errorf("write %s: %w", filename, err)
	}

	s.mu.Lock()
	s.knownHashes[hash] = true
	s.mu.Unlock()

	logger.Printf("Saved: %s (%d bytes)", filename, len(pngData))
	return nil
}

func generateFilename(dir string) string {
	now := time.Now()
	base := now.Format("2006-01-02_15-04-05")
	filename := base + ".png"

	// Handle collisions with suffix
	if _, err := os.Stat(filepath.Join(dir, filename)); err != nil {
		return filename
	}
	for i := 1; i < 100; i++ {
		filename = fmt.Sprintf("%s_%d.png", base, i)
		if _, err := os.Stat(filepath.Join(dir, filename)); err != nil {
			return filename
		}
	}
	// Fallback: include milliseconds
	return now.Format("2006-01-02_15-04-05.000") + ".png"
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
