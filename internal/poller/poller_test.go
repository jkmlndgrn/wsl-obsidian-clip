package poller

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeClipboard struct {
	data []byte
	err  error
}

func (f *fakeClipboard) Check() ([]byte, error) {
	return f.data, f.err
}

func (f *fakeClipboard) Close() error {
	return nil
}

type fakeEmbedWriter struct {
	filename string
	onPaste  func() error
	err      error
	offers   int
}

func (f *fakeEmbedWriter) OfferEmbed(filename string, onPaste func() error) error {
	f.offers++
	f.filename = filename
	f.onPaste = onPaste
	return f.err
}

func (f *fakeEmbedWriter) Close() error {
	return nil
}

func testLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestPollOffersNewImageAndSavesOnPaste(t *testing.T) {
	outputDir := t.TempDir()
	pngData := []byte("png data")
	state := &pollState{knownHashes: make(map[string]bool), outputDir: outputDir}
	embedWriter := &fakeEmbedWriter{}

	if err := poll(&fakeClipboard{data: pngData}, testLogger(), state, embedWriter); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if embedWriter.offers != 1 {
		t.Fatalf("offers = %d, want 1", embedWriter.offers)
	}
	if embedWriter.filename == "" || !strings.HasSuffix(embedWriter.filename, ".png") {
		t.Fatalf("unexpected filename %q", embedWriter.filename)
	}
	if embedWriter.onPaste == nil {
		t.Fatal("expected onPaste callback")
	}

	if err := embedWriter.onPaste(); err != nil {
		t.Fatalf("onPaste: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(outputDir, embedWriter.filename))
	if err != nil {
		t.Fatalf("read saved image: %v", err)
	}
	if string(got) != string(pngData) {
		t.Fatalf("saved data = %q, want %q", got, pngData)
	}
}

func TestPollDoesNotOfferDuplicateClipboardImage(t *testing.T) {
	state := &pollState{knownHashes: make(map[string]bool), outputDir: t.TempDir()}
	embedWriter := &fakeEmbedWriter{}
	clipboard := &fakeClipboard{data: []byte("same image")}

	if err := poll(clipboard, testLogger(), state, embedWriter); err != nil {
		t.Fatalf("first poll: %v", err)
	}
	if err := poll(clipboard, testLogger(), state, embedWriter); err != nil {
		t.Fatalf("second poll: %v", err)
	}
	if embedWriter.offers != 1 {
		t.Fatalf("offers = %d, want 1", embedWriter.offers)
	}
}

func TestPollResetsLastHashWhenClipboardHasNoImage(t *testing.T) {
	state := &pollState{knownHashes: make(map[string]bool), outputDir: t.TempDir()}
	embedWriter := &fakeEmbedWriter{}
	pngData := []byte("same image")

	if err := poll(&fakeClipboard{data: pngData}, testLogger(), state, embedWriter); err != nil {
		t.Fatalf("first image poll: %v", err)
	}
	if err := poll(&fakeClipboard{}, testLogger(), state, embedWriter); err != nil {
		t.Fatalf("empty poll: %v", err)
	}
	if err := poll(&fakeClipboard{data: pngData}, testLogger(), state, embedWriter); err != nil {
		t.Fatalf("second image poll: %v", err)
	}
	if embedWriter.offers != 2 {
		t.Fatalf("offers = %d, want 2", embedWriter.offers)
	}
}

func TestPollReportsClipboardAndOfferErrors(t *testing.T) {
	state := &pollState{knownHashes: make(map[string]bool), outputDir: t.TempDir()}

	_, clipboardErr := (&fakeClipboard{err: errors.New("clipboard failed")}).Check()
	if clipboardErr == nil {
		t.Fatal("test setup expected clipboard error")
	}
	if err := poll(&fakeClipboard{err: clipboardErr}, testLogger(), state, &fakeEmbedWriter{}); err == nil || !strings.Contains(err.Error(), "check clipboard") {
		t.Fatalf("expected clipboard error, got %v", err)
	}

	offerErr := errors.New("offer failed")
	if err := poll(&fakeClipboard{data: []byte("png")}, testLogger(), state, &fakeEmbedWriter{err: offerErr}); err == nil || !strings.Contains(err.Error(), "offer embed") {
		t.Fatalf("expected offer error, got %v", err)
	}
}

func TestIndexExistingRecordsPNGHashes(t *testing.T) {
	outputDir := t.TempDir()
	pngData := []byte("existing png")
	if err := os.WriteFile(filepath.Join(outputDir, "existing.png"), pngData, 0644); err != nil {
		t.Fatalf("write png: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "note.txt"), []byte("not indexed"), 0644); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	state := &pollState{knownHashes: make(map[string]bool), outputDir: outputDir}
	state.indexExisting(testLogger())

	if !state.knownHashes[hashBytes(pngData)] {
		t.Fatal("expected existing PNG hash to be indexed")
	}
	if len(state.knownHashes) != 1 {
		t.Fatalf("indexed hashes = %d, want 1", len(state.knownHashes))
	}
}

func TestGenerateFilenameAddsSuffixForCollision(t *testing.T) {
	dir := t.TempDir()
	first := generateFilename(dir)
	if err := os.WriteFile(filepath.Join(dir, first), []byte("existing"), 0644); err != nil {
		t.Fatalf("write collision file: %v", err)
	}

	second := generateFilename(dir)
	if second == first {
		t.Fatalf("expected collision suffix, got same filename %q", second)
	}
	if !strings.HasSuffix(second, "_1.png") {
		t.Fatalf("expected _1.png suffix, got %q", second)
	}
}
