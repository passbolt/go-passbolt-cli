package util

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
)

// Payloads an attacker can actually plant; Passbolt stores them verbatim.
func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "osc 52 clipboard write is neutralized",
			in:   "report\x1b]52;c;cGFzc2JvbHQtcG9j\x07",
			want: "report\uFFFD]52;c;cGFzc2JvbHQtcG9j\uFFFD",
		},
		{
			name: "csi erase line is neutralized",
			in:   "safe-name\x1b[2K\x1b[1A",
			want: "safe-name\uFFFD[2K\uFFFD[1A",
		},
		{
			name: "bare escape is replaced",
			in:   "a\x1bb",
			want: "a\uFFFDb",
		},
		{
			name: "8-bit c1 csi introducer is replaced",
			in:   "a\u009b2Kb",
			want: "a\uFFFD2Kb",
		},
		{
			name: "bidi override is replaced",
			in:   "invoice\u202egpj.exe",
			want: "invoice\uFFFDgpj.exe",
		},
		{
			name: "zero width space is replaced, so it cannot hide in a name",
			in:   "ad\u200bmin",
			want: "ad\uFFFDmin",
		},
		{
			name: "line and paragraph separators are replaced",
			in:   "a\u2028b\u2029c",
			want: "a\uFFFDb\uFFFDc",
		},
		{
			name: "carriage return is replaced so output cannot be overwritten",
			in:   "real\rfake",
			want: "real\uFFFDfake",
		},
		{
			name: "newline is replaced so a name cannot forge output lines",
			in:   "real-name\nPermissions: owner",
			want: "real-name\uFFFDPermissions: owner",
		},
		{
			name: "tab is replaced so a name cannot forge table columns",
			in:   "a\tb",
			want: "a\uFFFDb",
		},
		{
			name: "raw c1 byte is replaced, not passed through as invalid utf-8",
			in:   "a\x9b[31mb",
			want: "a\uFFFD[31mb",
		},
		{
			name: "truncated utf-8 sequence is replaced",
			in:   "caf\xc3",
			want: "caf\uFFFD",
		},
		{
			name: "one replacement per byte of a broken sequence",
			in:   "a\xed\xa0\x80b",
			want: "a\uFFFD\uFFFD\uFFFDb",
		},
		{
			name: "encoded replacement char is kept",
			in:   "a\uFFFDb",
			want: "a\uFFFDb",
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

// Sanitized at the call site and again in PrintTable: one U+FFFD must not become two.
func TestSanitizeIsIdempotent(t *testing.T) {
	in := "name\x1b]52;c;AAAA\x07\x1b[2K"
	once := Sanitize(in)
	twice := Sanitize(once)
	if twice != once {
		t.Errorf("not idempotent: %q then %q", once, twice)
	}
	if got, want := strings.Count(twice, string(replacementChar)), strings.Count(once, string(replacementChar)); got != want {
		t.Errorf("replacement count grew: %d, want %d", got, want)
	}
}

// Why this is a writer, not a per-call strings.Map: a rune can straddle two Writes.
func TestSanitizingWriterSplitRune(t *testing.T) {
	full := []byte("Café \x1b]52;c;AAAA\x07 ok\n")
	want := "Café \uFFFD]52;c;AAAA\uFFFD ok\n"

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
		// A split inside the é must not leave it behind as two broken bytes.
		if n := strings.Count(got.String(), string(replacementChar)); n != 2 {
			t.Errorf("split at %d: %d replacements, want 2: %q", split, n, got.String())
		}
	}
}

// A decrypted V4 secret is raw bytes, so the stream can carry invalid UTF-8.
func TestSanitizingWriterReplacesInvalidBytes(t *testing.T) {
	var got bytes.Buffer
	w := NewSanitizingWriter(&got)
	if _, err := w.Write([]byte("a\x9b[31m\xed\xa0\x80b\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if want := "a\uFFFD[31m\uFFFD\uFFFD\uFFFDb\n"; got.String() != want {
		t.Errorf("got %q, want %q", got.String(), want)
	}
}

// fmt treats a short write as an error, and here the write is longer than the input.
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
	if want := "a\uFFFD\uFFFDb"; got.String() != want {
		t.Errorf("got %q, want %q", got.String(), want)
	}
	if got.Len() <= len(in) {
		t.Errorf("expected the replacement to outgrow the input: %d vs %d", got.Len(), len(in))
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

// A stream ending inside a character would otherwise lose those bytes silently.
func TestSanitizingWriterFlushesPartial(t *testing.T) {
	var got bytes.Buffer
	w := NewSanitizingWriter(&got)
	if _, err := w.Write([]byte("caf\xc3")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got.String() != "caf" {
		t.Errorf("before flush: got %q, want %q", got.String(), "caf")
	}

	flusher, ok := w.(interface{ Flush() error })
	if !ok {
		t.Fatal("the sanitizing writer no longer flushes")
	}
	if err := flusher.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if want := "caf\uFFFD"; got.String() != want {
		t.Errorf("after flush: got %q, want %q", got.String(), want)
	}
	// Nothing is held back any more, so a second flush must add nothing.
	if err := flusher.Flush(); err != nil {
		t.Fatalf("second flush: %v", err)
	}
	if want := "caf\uFFFD"; got.String() != want {
		t.Errorf("after second flush: got %q, want %q", got.String(), want)
	}
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
	if want := "Name: grp\uFFFD]52;c;AAAA\uFFFD\uFFFD[2K\n"; got.String() != want {
		t.Errorf("got %q, want %q", got.String(), want)
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
	if want := "line one\nline two\tcol\uFFFD]52;c;AAAA\uFFFD\uFFFD"; got.String() != want {
		t.Errorf("got %q, want %q", got.String(), want)
	}
}

// The user is told once, at the end of the run.
func TestWarnReplacements(t *testing.T) {
	var got bytes.Buffer
	orig := Err
	Err = NewSanitizingWriter(&got)
	t.Cleanup(func() { Err = orig; replacedRunes.Store(0) })

	replacedRunes.Store(0)
	WarnReplacements()
	if got.Len() != 0 {
		t.Errorf("warned with nothing replaced: %q", got.String())
	}

	Sanitize("a\x1bb\x1bc")
	WarnReplacements()
	if want := "Warning: replaced 2 unprintable characters with U+FFFD (\uFFFD)\n"; got.String() != want {
		t.Errorf("got %q, want %q", got.String(), want)
	}

	// The count is spent, and the printable warning cannot warn about itself.
	before := got.Len()
	WarnReplacements()
	if got.Len() != before {
		t.Errorf("warned twice: %q", got.String()[before:])
	}
}

// The singular case, because the count is in the message.
func TestWarnReplacementsSingular(t *testing.T) {
	var got bytes.Buffer
	orig := Err
	Err = NewSanitizingWriter(&got)
	t.Cleanup(func() { Err = orig; replacedRunes.Store(0) })

	replacedRunes.Store(0)
	Sanitize("a\x1bb")
	WarnReplacements()
	if want := "1 unprintable character with"; !strings.Contains(got.String(), want) {
		t.Errorf("got %q, want it to contain %q", got.String(), want)
	}
}

// What Execute calls before warning: the tail of a truncated stream is counted too.
func TestFlushOutputs(t *testing.T) {
	var out, errBuf bytes.Buffer
	origOut, origErr := Out, Err
	Out, Err = NewSanitizingWriter(&out), NewSanitizingWriter(&errBuf)
	t.Cleanup(func() { Out, Err = origOut, origErr; replacedRunes.Store(0) })

	replacedRunes.Store(0)
	Print("caf\xc3")
	FlushOutputs()
	if want := "caf\uFFFD"; out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
	if got := replacedRunes.Load(); got != 1 {
		t.Errorf("counted %d replacements, want 1", got)
	}
}
