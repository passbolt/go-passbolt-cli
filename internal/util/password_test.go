package util

import (
	"os"
	"testing"
)

// Test stdin is a pipe, never a terminal, so these tests cover the
// non-interactive paths of ReadPassword and ReadNewPassword. The interactive
// masked input needs a real TTY and is exercised manually.

func withStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("write: %v", err)
	}
	w.Close()
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = old
		r.Close()
	})
}

func TestReadPassword_Pipe(t *testing.T) {
	withStdin(t, "secret\n")
	got, err := ReadPassword("prompt:")
	if err != nil {
		t.Fatalf("ReadPassword: %v", err)
	}
	if got != "secret" {
		t.Errorf("ReadPassword = %q, want secret", got)
	}
}

func TestReadNewPassword_Pipe(t *testing.T) {
	// On a pipe there is no confirmation round-trip: one line, non-empty.
	withStdin(t, "kdbx-pass\n")
	got, err := ReadNewPassword("p:", "c:")
	if err != nil {
		t.Fatalf("ReadNewPassword: %v", err)
	}
	if got != "kdbx-pass" {
		t.Errorf("ReadNewPassword = %q, want kdbx-pass", got)
	}
}

func TestReadNewPassword_PipeRejectsEmpty(t *testing.T) {
	withStdin(t, "\n")
	if got, err := ReadNewPassword("p:", "c:"); err == nil {
		t.Errorf("empty password accepted: %q, want error", got)
	}
}
