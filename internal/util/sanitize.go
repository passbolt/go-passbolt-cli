package util

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
)

// Out and Err wrap stdout and stderr, so anything printed to the terminal is cleaned
// on the way out. Raw output (--json, completions, subprocess stdio) skips them.
var (
	Out io.Writer = NewSanitizingWriter(os.Stdout)
	Err io.Writer = NewSanitizingWriter(os.Stderr)
)

// Removing characters hid the change: "ad\u200bmin" rendered as "admin".
const replacementChar = '\uFFFD'

// Feeds the warning at the end of the run. Atomic: the worker pool prints too.
var replacedRunes atomic.Int64

func Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(Out, format, args...)
}

func Println(args ...any) {
	_, _ = fmt.Fprintln(Out, args...)
}

func Print(args ...any) {
	_, _ = fmt.Fprint(Out, args...)
}

func Eprintf(format string, args ...any) {
	_, _ = fmt.Fprintf(Err, format, args...)
}

func Eprintln(args ...any) {
	_, _ = fmt.Fprintln(Err, args...)
}

func Eprint(args ...any) {
	_, _ = fmt.Fprint(Err, args...)
}

// WarnReplacements says, once, that this is not the text as stored. Execute calls it
// after the command; from Write it would land between two table rows.
func WarnReplacements() {
	replaced := replacedRunes.Swap(0)
	if replaced == 0 {
		return
	}
	noun := "characters"
	if replaced == 1 {
		noun = "character"
	}
	Eprintf("Warning: replaced %d unprintable %s with U+FFFD (%c)\n", replaced, noun, replacementChar)
}

// Exit ends the process the way Execute does. Every exit goes through here, or
// output printed just before it leaves without its warning.
func Exit(code int) {
	FlushOutputs()
	WarnReplacements()
	os.Exit(code)
}

// FlushOutputs releases what Out and Err held back mid-character. By assertion:
// both are typed io.Writer.
func FlushOutputs() {
	for _, writer := range []io.Writer{Out, Err} {
		if flusher, ok := writer.(interface{ Flush() error }); ok {
			_ = flusher.Flush()
		}
	}
}

// Sanitize prepares a server value for a single line of output. On top of what the
// writer does, it replaces \n and \t, which could fake a line or break a table row.
func Sanitize(text string) string {
	var kept strings.Builder
	kept.Grow(len(text))
	replaced := 0
	for pos := 0; pos < len(text); {
		char, width := utf8.DecodeRuneInString(text[pos:])
		if keepChar(char, width, false) {
			kept.WriteString(text[pos : pos+width])
		} else {
			kept.WriteRune(replacementChar)
			replaced++
		}
		pos += width
	}
	replacedRunes.Add(int64(replaced))
	return kept.String()
}

// keepChar says whether a character may reach the terminal as itself; the rest is
// replaced, or escaped on the JSON path. Only the stream keeps line breaks.
func keepChar(char rune, width int, keepLineBreaks bool) bool {
	// A width of 1 means a broken byte rather than a real U+FFFD character. Some
	// terminals read such a byte as a control code, so it does not pass as itself.
	if char == utf8.RuneError && width == 1 {
		return false
	}
	if keepLineBreaks && (char == '\n' || char == '\t') {
		return true
	}
	// Keeps letters, digits, punctuation and the plain space. Rejects control codes
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
	// byte by byte (io.Copy) would replace every multi-byte character with U+FFFD.
	remaining := input
	if len(writer.partial) > 0 {
		remaining = slices.Concat(writer.partial, input)
		writer.partial = nil
	}

	// FullRune stops the loop on a character split across two Writes, never on a
	// broken byte, which counts as complete and is replaced below. kept can outgrow
	// the hint: U+FFFD is 3 bytes where the byte it stands for was 1.
	kept := make([]byte, 0, len(remaining))
	replaced := 0
	for len(remaining) > 0 && utf8.FullRune(remaining) {
		char, width := utf8.DecodeRune(remaining)
		if keepChar(char, width, true) {
			kept = append(kept, remaining[:width]...)
		} else {
			// One per byte of a broken sequence, as string([]byte) does.
			kept = utf8.AppendRune(kept, replacementChar)
			replaced++
		}
		remaining = remaining[width:]
	}
	writer.partial = append(writer.partial, remaining...)

	if _, err := writer.dest.Write(kept); err != nil {
		return 0, err
	}
	replacedRunes.Add(int64(replaced))

	// fmt reads a short count as a failed write, and replaced bytes were consumed.
	return len(input), nil
}

// Flush replaces whatever the last Write held back, which a stream ending inside a
// character would otherwise lose without a trace.
func (writer *sanitizingWriter) Flush() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if len(writer.partial) == 0 {
		return nil
	}
	kept := make([]byte, 0, len(writer.partial)*utf8.RuneLen(replacementChar))
	replaced := len(writer.partial)
	for range replaced {
		kept = utf8.AppendRune(kept, replacementChar)
	}
	writer.partial = nil

	if _, err := writer.dest.Write(kept); err != nil {
		return err
	}
	replacedRunes.Add(int64(replaced))
	return nil
}
