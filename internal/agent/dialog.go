package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"
)

var ErrDenied = errors.New("agent approval denied or canceled")
var errNoDesktop = errors.New("desktop dialog unavailable")

type Dialog struct {
	Backend     string
	Pinentry    string
	Terminal    string
	PID         int
	Environment []string
}

// Text removes terminal controls and bounds untrusted server/command descriptions.
func Text(s string) string { return boundedText(s, 1024) }

// dialogText allows complete resource names and identity details in approvals.
func dialogText(s string) string { return boundedText(s, 16384) }

func boundedText(s string, limit int) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('\uFFFD')
		}
		if b.Len() > limit {
			break
		}
	}
	return b.String()
}
func (d Dialog) Ask(ctx context.Context, title, prompt string, secret bool) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	title, prompt = dialogText(title), dialogText(prompt)
	switch d.Backend {
	case "pinentry":
		return d.pinentry(ctx, title, prompt, secret)
	case "terminal":
		return d.terminal(ctx, title, prompt, secret)
	case "", "auto":
		value, err := d.desktop(ctx, title, prompt, secret)
		if !errors.Is(err, errNoDesktop) {
			return value, err
		}
		return d.terminal(ctx, title, prompt, secret)
	default:
		return "", fmt.Errorf("unknown agent dialog backend %q", d.Backend)
	}
}
func (d Dialog) Confirm(ctx context.Context, title, prompt string) error {
	_, err := d.Ask(ctx, title, prompt, false)
	return err
}

func (d Dialog) terminal(ctx context.Context, title, prompt string, secret bool) (string, error) {
	in, out, cleanup, err := d.openTerminal()
	if err != nil {
		return "", fmt.Errorf("no dialog available: select pinentry, install a desktop helper, or use a controlling terminal: %w", err)
	}
	defer cleanup()
	state, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(int(in.Fd()), state) }()
	_, err = io.WriteString(out, fmt.Sprintf("\x1b[?1049h\x1b[2J\x1b[H%s\r\n\r\n%s\r\n\r\n", title, prompt))
	if err != nil {
		return "", err
	}
	defer func() { _, _ = io.WriteString(out, "\x1b[?1049l") }()
	if secret {
		_, err = io.WriteString(out, "Enter value (hidden), Esc to cancel: ")
	} else {
		_, err = io.WriteString(out, "[y] Allow  [n/Esc] Deny (default)\r\n")
	}
	if err != nil {
		return "", err
	}
	var value []byte
	defer func() { clear(value) }()
	var b [4]byte
	defer clear(b[:])
	for {
		n, err := terminalBytes(ctx, in, b[:])
		if err != nil {
			return "", err
		}
		switch b[0] {
		case 3, 27:
			return "", ErrDenied
		case '\r', '\n':
			if secret {
				return string(value), nil
			}
			return "", ErrDenied
		case 127, 8:
			if len(value) > 0 {
				_, width := utf8.DecodeLastRune(value)
				clear(value[len(value)-width:])
				value = value[:len(value)-width]
			}
		default:
			if secret {
				if len(value)+n > 4096 {
					return "", fmt.Errorf("input too long")
				}
				value = append(value, b[:n]...)
			} else {
				if b[0] == 'y' || b[0] == 'Y' {
					return "", nil
				}
				return "", ErrDenied
			}
		}
	}
}
func assuanEscape(s string) string {
	return strings.NewReplacer("%", "%25", "\n", "%0A", "\r", "%0D").Replace(s)
}
func readAssuan(r io.Reader) (string, error) {
	// Callers supply one shared buffered reader, so buffered lines are not lost.
	reader, ok := r.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(r)
	}
	var value strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		if len(line) > 65536 {
			return "", fmt.Errorf("pinentry response too long")
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "OK" || strings.HasPrefix(line, "OK ") {
			return value.String(), nil
		}
		if strings.HasPrefix(line, "ERR") {
			return "", ErrDenied
		}
		if strings.HasPrefix(line, "D ") {
			decoded, err := url.PathUnescape(strings.TrimPrefix(line, "D "))
			if err != nil {
				return "", err
			}
			value.WriteString(decoded)
		} else if !strings.HasPrefix(line, "S ") && !strings.HasPrefix(line, "#") {
			return "", fmt.Errorf("unexpected pinentry response")
		}
	}
}
func (d Dialog) pinentry(ctx context.Context, title, prompt string, secret bool) (string, error) {
	binary := d.Pinentry
	if binary == "" {
		binary = "pinentry"
	}
	cmd := exec.CommandContext(ctx, binary)
	cmd.Env = d.Environment
	in, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err = cmd.Start(); err != nil {
		return "", err
	}
	defer func() { _ = in.Close(); _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	reader := bufio.NewReader(out)
	if _, err = readAssuan(reader); err != nil {
		return "", err
	}
	commands := []string{"SETTITLE " + assuanEscape(title), "SETDESC " + assuanEscape(prompt), "SETOK Allow", "SETCANCEL Deny"}
	if d.Terminal != "" {
		commands = append(commands, "OPTION ttyname="+assuanEscape(d.Terminal))
	}
	for _, entry := range d.Environment {
		if strings.HasPrefix(entry, "TERM=") {
			commands = append(commands, "OPTION ttytype="+assuanEscape(strings.TrimPrefix(entry, "TERM=")))
		}
	}
	for _, command := range commands {
		if _, err = io.WriteString(in, command+"\n"); err != nil {
			return "", err
		}
		if _, err = readAssuan(reader); err != nil {
			return "", err
		}
	}
	operation := "CONFIRM"
	if secret {
		operation = "GETPIN"
	}
	if _, err = io.WriteString(in, operation+"\n"); err != nil {
		return "", err
	}
	return readAssuan(reader)
}

// RunDialog executes fixed scripts or helper arguments, never shell-interpolated text.
func (d Dialog) run(ctx context.Context, binary string, args []string, input string) (string, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = d.Environment
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", errNoDesktop
		}
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			return "", ErrDenied
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("dialog helper failed")
	}
	return strings.TrimSuffix(strings.TrimSuffix(string(out), "\n"), "\r"), nil
}
