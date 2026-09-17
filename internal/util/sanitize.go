package util

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Out and Err wrap stdout and stderr, so anything printed to the terminal is cleaned
// on the way out. Raw output (--json, completions, subprocess stdio) skips them.
var (
	Out io.Writer = NewSanitizingWriter(os.Stdout)
	Err io.Writer = NewSanitizingWriter(os.Stderr)
)

func Printf(format string, args ...any) { _, _ = fmt.Fprintf(Out, format, args...) }

func Println(args ...any) { _, _ = fmt.Fprintln(Out, args...) }

func Print(args ...any) { _, _ = fmt.Fprint(Out, args...) }

func Eprintf(format string, args ...any) { _, _ = fmt.Fprintf(Err, format, args...) }

func Eprintln(args ...any) { _, _ = fmt.Fprintln(Err, args...) }

func Eprint(args ...any) { _, _ = fmt.Fprint(Err, args...) }

// Sanitize prepares a server value for a single line of output. On top of what the
// writer does, it removes \n and \t, which could fake a line or break a table row.
func Sanitize(text string) string {
	var kept strings.Builder
	kept.Grow(len(text))
	for pos := 0; pos < len(text); {
		char, width := utf8.DecodeRuneInString(text[pos:])
		if keepChar(char, width, false) {
			kept.WriteString(text[pos : pos+width])
		}
		pos += width
	}
	return kept.String()
}

// keepChar says whether one character may reach the terminal. Only the stream keeps
// line breaks; a value printed on one line must not.
func keepChar(char rune, width int, keepLineBreaks bool) bool {
	// A width of 1 means a broken byte rather than a real U+FFFD character. Some
	// terminals read such a byte as a control code, so drop it.
	if char == utf8.RuneError && width == 1 {
		return false
	}
	if keepLineBreaks && (char == '\n' || char == '\t') {
		return true
	}
	// Keeps letters, digits, punctuation and the plain space. Drops control codes
	// and invisible characters such as zero-width spaces and bidi overrides.
	return unicode.IsPrint(char)
}

type sanitizingWriter struct {
	mu      sync.Mutex
	dest    io.Writer
	partial []byte
}

// NewSanitizingWriter wraps dest to filter what is written to it. Out and Err are
// built with it, so nothing reaches stdout or stderr unfiltered.
func NewSanitizingWriter(dest io.Writer) io.Writer { return &sanitizingWriter{dest: dest} }

func (writer *sanitizingWriter) Write(input []byte) (int, error) {
	// Out and Err are globals, so two goroutines can print at once.
	writer.mu.Lock()
	defer writer.mu.Unlock()

	// Whatever the last Write held back comes first. Without this, a caller writing
	// byte by byte (io.Copy) would lose every multi-byte character as a broken byte.
	remaining := input
	if len(writer.partial) > 0 {
		remaining = slices.Concat(writer.partial, input)
		writer.partial = nil
	}

	// FullRune stops the loop on a character split across two Writes, never on a
	// broken byte, which counts as complete and is dropped by keepChar.
	kept := make([]byte, 0, len(remaining))
	for len(remaining) > 0 && utf8.FullRune(remaining) {
		char, width := utf8.DecodeRune(remaining)
		if keepChar(char, width, true) {
			kept = append(kept, remaining[:width]...)
		}
		remaining = remaining[width:]
	}
	writer.partial = append(writer.partial, remaining...)

	if _, err := writer.dest.Write(kept); err != nil {
		return 0, err
	}
	// fmt reads a count smaller than the input as a failed write.
	return len(input), nil
}
