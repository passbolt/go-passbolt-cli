package util

import (
	"net/http"
	"testing"

	"github.com/spf13/viper"
)

func TestGetExtraHeaders(t *testing.T) {
	t.Cleanup(func() { viper.Set("headers", nil) })

	tests := []struct {
		name    string
		entries []string
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "empty",
			entries: nil,
			want:    nil,
		},
		{
			name:    "cookie and trimmed value",
			entries: []string{"Cookie: _oauth2_proxy=abc123"},
			want:    map[string]string{"Cookie": "_oauth2_proxy=abc123"},
		},
		{
			name:    "value containing colons is preserved",
			entries: []string{"X-Token: a:b:c"},
			want:    map[string]string{"X-Token": "a:b:c"},
		},
		{
			name:    "multiple headers",
			entries: []string{"CF-Access-Client-Id: id", "CF-Access-Client-Secret: secret"},
			want: map[string]string{
				"CF-Access-Client-Id":     "id",
				"CF-Access-Client-Secret": "secret",
			},
		},
		{
			name:    "missing colon errors",
			entries: []string{"NotAHeader"},
			wantErr: true,
		},
		{
			name:    "empty name errors",
			entries: []string{": value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("headers", tt.entries)
			got, err := GetExtraHeaders()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Fatalf("header %q = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// recordingRoundTripper captures the request it receives so the test can assert
// on the headers the injector produced.
type recordingRoundTripper struct {
	gotHeader http.Header
}

func (r *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.gotHeader = req.Header.Clone()
	return &http.Response{StatusCode: 200, Body: http.NoBody, Header: make(http.Header)}, nil
}

func TestHeaderRoundTripperAppendsCookie(t *testing.T) {
	rec := &recordingRoundTripper{}
	rt := &headerRoundTripper{
		base:    rec,
		headers: map[string]string{"Cookie": "_oauth2_proxy=abc123"},
	}

	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the Passbolt SDK having already set its own session cookies.
	req.Header.Set("Cookie", "passbolt_session=xyz")

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	got := rec.gotHeader.Get("Cookie")
	want := "passbolt_session=xyz; _oauth2_proxy=abc123"
	if got != want {
		t.Fatalf("Cookie = %q, want %q", got, want)
	}

	// The original request must not be mutated by the RoundTripper.
	if orig := req.Header.Get("Cookie"); orig != "passbolt_session=xyz" {
		t.Fatalf("original request mutated, Cookie = %q", orig)
	}
}

func TestHeaderRoundTripperSetsArbitraryHeader(t *testing.T) {
	rec := &recordingRoundTripper{}
	rt := &headerRoundTripper{
		base:    rec,
		headers: map[string]string{"CF-Access-Client-Id": "id"},
	}

	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	if got := rec.gotHeader.Get("CF-Access-Client-Id"); got != "id" {
		t.Fatalf("CF-Access-Client-Id = %q, want %q", got, "id")
	}
}
