package clipboard

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"io"
	"log"
	"strings"
	"testing"
)

type writeCloser struct {
	bytes.Buffer
	closed bool
}

func (w *writeCloser) Close() error {
	w.closed = true
	return nil
}

func newTestClient(response string) (*Client, *writeCloser) {
	stdin := &writeCloser{}
	scanner := bufio.NewScanner(strings.NewReader(response))
	return &Client{
		stdin:  stdin,
		stdout: scanner,
		logger: log.New(io.Discard, "", 0),
	}, stdin
}

func TestCheckReturnsNilForNone(t *testing.T) {
	client, stdin := newTestClient("NONE\n")

	data, err := client.Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if data != nil {
		t.Fatalf("expected nil data, got %q", data)
	}
	if got := strings.TrimSpace(stdin.String()); got != "CHECK" {
		t.Fatalf("stdin = %q, want CHECK", got)
	}
}

func TestCheckReturnsDecodedImage(t *testing.T) {
	want := []byte("png bytes")
	response := "IMAGE\n" + base64.StdEncoding.EncodeToString(want) + "\nEND\n"
	client, _ := newTestClient(response)

	got, err := client.Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Check() = %q, want %q", got, want)
	}
}

func TestCheckReportsInvalidImageResponse(t *testing.T) {
	testCases := []struct {
		name     string
		response string
		wantErr  string
	}{
		{
			name:     "invalid base64",
			response: "IMAGE\nnot-base64\nEND\n",
			wantErr:  "decode base64",
		},
		{
			name:     "missing end marker",
			response: "IMAGE\n" + base64.StdEncoding.EncodeToString([]byte("png")) + "\nDONE\n",
			wantErr:  "expected END",
		},
		{
			name:     "unexpected response",
			response: "WHAT\n",
			wantErr:  "unexpected response",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := newTestClient(tc.response)

			_, err := client.Check()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
