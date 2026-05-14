package cmd

import (
	"bytes"
	"errors"
	"log"
	"testing"
)

func TestCheckPlatformQuietRestoresDefaultLogger(t *testing.T) {
	var buf bytes.Buffer
	originalOutput := log.Default().Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
	})

	oldPlatformCheck := platformCheck
	t.Cleanup(func() {
		platformCheck = oldPlatformCheck
	})

	platformCheck = func() error {
		log.Print("xgb noise")
		return errors.New("display failed")
	}

	err := checkPlatform(true)
	if err == nil || err.Error() != "display failed" {
		t.Fatalf("expected platform error, got %v", err)
	}
	if buf.String() != "" {
		t.Fatalf("expected quiet platform check to suppress logs, got %q", buf.String())
	}

	log.Print("restored")
	if buf.String() == "" {
		t.Fatal("expected default logger output to be restored")
	}
}
