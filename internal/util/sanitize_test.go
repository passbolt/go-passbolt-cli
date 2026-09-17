package util

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"
)

// Payloads an attacker can actually plant; Passbolt stores them verbatim (PB-54765).
func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "osc 52 clipboard write is neutralized",
			in:   "report\x1b]52;c;cGFzc2JvbHQtcG9j\x07",
			want: "report]52;c;cGFzc2JvbHQtcG9j",
		},
		{
			name: "csi erase line is neutralized",
			in:   "safe-name\x1b[2K\x1b[1A",
			want: "safe-name[2K[1A",
		},
		{
			name: "bare escape is dropped",
			in:   "a\x1bb",
			want: "ab",
		},
		{
			name: "8-bit c1 csi introducer is dropped",
			in:   "a\u009b2Kb",
			want: "a2Kb",
		},
		{
			name: "bidi override is dropped",
			in:   "invoice\u202egpj.exe",
			want: "invoicegpj.exe",
		},
		{
			name: "zero width space is dropped",
			in:   "ad\u200bmin",
			want: "admin",
		},
		{
			name: "line and paragraph separators are dropped",
			in:   "a\u2028b\u2029c",
			want: "abc",
		},
		{
			name: "carriage return is dropped so output cannot be overwritten",
			in:   "real\rfake",
			want: "realfake",
		},
		{
			name: "newline is dropped so a name cannot forge output lines",
			in:   "real-name\nPermissions: owner",
			want: "real-namePermissions: owner",
		},
		{
			name: "tab is dropped so a name cannot forge table columns",
			in:   "a\tb",
			want: "ab",
		},
		{
			name: "raw c1 byte is dropped, not passed through as invalid utf-8",
			in:   "a\x9b[31mb",
			want: "a[31mb",
		},
		{
			name: "truncated utf-8 sequence is dropped",
			in:   "caf\xc3",
			want: "caf",
		},
		{
			name: "encoded replacement char is kept",
			in:   "a�b",
			want: "a�b",
		},
		{
			name: "ordinary text is untouched",
			in:   "Staging DB (eu-west-1)",
			want: "Staging DB (eu-west-1)",
		},
		{
			name: "non-ascii printable text is untouched",
			in:   "Café Ünïcode 密码",
			want: "Café Ünïcode 密码",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(tt.in); got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Values are sanitized at the call site and again centrally in PrintTable.
func TestSanitizeIsIdempotent(t *testing.T) {
	in := "name\x1b]52;c;AAAA\x07\x1b[2K"
	once := Sanitize(in)
	if twice := Sanitize(once); twice != once {
		t.Errorf("not idempotent: %q then %q", once, twice)
	}
}

// Why this is a writer, not a per-call strings.Map: a rune can straddle two Writes.
func TestSanitizingWriterSplitRune(t *testing.T) {
	full := []byte("Café \x1b]52;c;AAAA\x07 ok\n")
	want := "Café ]52;c;AAAA ok\n"

	for split := 1; split < len(full); split++ {
		var got bytes.Buffer
		w := NewSanitizingWriter(&got)
		if _, err := w.Write(full[:split]); err != nil {
			t.Fatalf("split %d: first write: %v", split, err)
		}
		if _, err := w.Write(full[split:]); err != nil {
			t.Fatalf("split %d: second write: %v", split, err)
		}
		if got.String() != want {
			t.Errorf("split at %d: got %q, want %q", split, got.String(), want)
		}
	}
}

// A decrypted V4 secret is raw bytes, so the stream can carry invalid UTF-8.
func TestSanitizingWriterDropsInvalidBytes(t *testing.T) {
	var got bytes.Buffer
	w := NewSanitizingWriter(&got)
	if _, err := w.Write([]byte("a\x9b[31m\xed\xa0\x80b\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got.String() != "a[31mb\n" {
		t.Errorf("got %q, want %q", got.String(), "a[31mb\n")
	}
}

// fmt treats a short write as an error; dropped bytes were still consumed.
func TestSanitizingWriterReportsFullLength(t *testing.T) {
	var got bytes.Buffer
	w := NewSanitizingWriter(&got)
	in := []byte("a\x1b\x07b")
	n, err := w.Write(in)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if n != len(in) {
		t.Errorf("n = %d, want %d", n, len(in))
	}
	if got.String() != "ab" {
		t.Errorf("got %q, want %q", got.String(), "ab")
	}
}

// Guards the partial-rune buffer against the decryption worker pool.
func TestSanitizingWriterConcurrent(t *testing.T) {
	w := NewSanitizingWriter(io.Discard)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 200 {
				if _, err := fmt.Fprintf(w, "Café \x1b]52;c;AAAA\x07 line\n"); err != nil {
					t.Errorf("write: %v", err)
					return
				}
			}
		})
	}
	wg.Wait()
}

// The contract forbidigo enforces: util.Printf never leaks a control byte.
func TestPrintfGoesThroughSanitizer(t *testing.T) {
	var got bytes.Buffer
	orig := Out
	Out = NewSanitizingWriter(&got)
	t.Cleanup(func() { Out = orig })

	Printf("Name: %s\n", "grp\x1b]52;c;AAAA\x07\x1b[2K")
	if bytes.ContainsAny(got.Bytes(), "\x1b\x07") {
		t.Errorf("control byte reached the stream: %q", got.String())
	}
	if got.String() != "Name: grp]52;c;AAAA[2K\n" {
		t.Errorf("got %q", got.String())
	}
}

// Without \n and \t the output collapses onto one line. A description is printed
// without Sanitize, so the writer is the only filter it meets.
func TestSanitizingWriterKeepsNewlinesAndTabs(t *testing.T) {
	var got bytes.Buffer
	w := NewSanitizingWriter(&got)
	if _, err := w.Write([]byte("line one\nline two\tcol\x1b]52;c;AAAA\x07\r")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if want := "line one\nline two\tcol]52;c;AAAA"; got.String() != want {
		t.Errorf("got %q, want %q", got.String(), want)
	}
}
