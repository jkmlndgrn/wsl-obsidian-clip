package embed

import (
	"testing"
	"time"
)

func TestShouldSaveForRequestor(t *testing.T) {
	testCases := []struct {
		name      string
		requestor requestorInfo
		age       time.Duration
		want      bool
	}{
		{
			name: "obsidian paste target saves immediately",
			requestor: requestorInfo{
				class: "obsidian Obsidian",
				name:  "Notes",
			},
			age:  time.Second,
			want: true,
		},
		{
			name: "chromium clipboard saves after probe window",
			requestor: requestorInfo{
				name: "Chromium clipboard",
			},
			age:  time.Second,
			want: true,
		},
		{
			name: "clipboard manager class does not save",
			requestor: requestorInfo{
				class: "copyq CopyQ",
			},
			age:  time.Second,
			want: false,
		},
		{
			name: "clipboard manager window name does not save",
			requestor: requestorInfo{
				name: "Clipboard Indicator",
			},
			age:  time.Second,
			want: false,
		},
		{
			name: "unknown requestor does not save early",
			requestor: requestorInfo{
			},
			age:  time.Second,
			want: false,
		},
		{
			name: "unknown requestor saves after timeout",
			requestor: requestorInfo{
			},
			age:  3 * time.Second,
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldSaveForRequestor(tc.requestor, tc.age)
			if got != tc.want {
				t.Fatalf("shouldSaveForRequestor(%+v, %s) = %v, want %v", tc.requestor, tc.age, got, tc.want)
			}
		})
	}
}

func TestNormalizeNullSeparated(t *testing.T) {
	got := normalizeNullSeparated([]byte("obsidian\x00Obsidian\x00"))
	if got != "obsidian Obsidian" {
		t.Fatalf("normalizeNullSeparated() = %q, want %q", got, "obsidian Obsidian")
	}
}
