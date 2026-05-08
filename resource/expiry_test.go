package resource

import (
	"testing"
	"time"
)

func TestParseExpiry_Absolute(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // RFC3339 in UTC
	}{
		{"rfc3339 utc", "2030-01-01T00:00:00Z", "2030-01-01T00:00:00Z"},
		{"rfc3339 with offset normalises to utc", "2030-01-01T00:00:00+02:00", "2029-12-31T22:00:00Z"},
		{"rfc3339 nano accepted", "2030-01-01T00:00:00.123456789Z", "2030-01-01T00:00:00Z"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseExpiry(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ParseExpiry(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseExpiry_Duration(t *testing.T) {
	// Duration inputs are parsed via time.ParseDuration, so only h/m/s units
	// are accepted — d and w are not supported despite the flag's help text
	// suggesting otherwise. This locks in the actual behavior; if the CLI
	// grows custom day/week parsing this test will fail and prompt updating.
	before := time.Now().UTC()
	got, err := ParseExpiry("48h")
	if err != nil {
		t.Fatalf("ParseExpiry(48h) errored: %v", err)
	}
	after := time.Now().UTC()
	gotT, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("returned value not RFC3339: %q", got)
	}
	if d := gotT.Sub(before); d < 47*time.Hour || d > 49*time.Hour {
		t.Errorf("ParseExpiry(48h) yielded %v, expected ~48h from now (%v..%v)", gotT, before, after)
	}
}

func TestParseExpiry_Errors(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"7d not supported by time.ParseDuration", "7d"},
		{"1w2d3h not supported", "1w2d3h"},
		{"completely invalid", "tomorrow"},
		{"non-rfc3339 timestamp", "2030/01/01"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseExpiry(tc.in); err == nil {
				t.Errorf("ParseExpiry(%q) expected error, got nil", tc.in)
			}
		})
	}
}

func TestParseExpiry_Empty(t *testing.T) {
	got, err := ParseExpiry("")
	if err != nil {
		t.Fatalf("empty input errored: %v", err)
	}
	if got != "" {
		t.Errorf("empty input should return empty, got %q", got)
	}
}

func TestIsUUID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"f848277c-5398-58f8-a82a-72397af2d450", true},
		{"F848277C-5398-58F8-A82A-72397AF2D450", true}, // case-insensitive
		{"00000000-0000-0000-0000-000000000000", true},
		{"not-a-uuid", false},
		{"", false},
		{"f848277c-5398-58f8-a82a-72397af2d45", false},   // 11-char tail
		{"f848277c-5398-58f8-a82a-72397af2d4500", false}, // 13-char tail
		{"f848277c_5398_58f8_a82a_72397af2d450", false},  // wrong separators
		{"f848277c-5398-58f8-a82a-72397af2d450-extra", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := isUUID(tc.in); got != tc.want {
				t.Errorf("isUUID(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
