package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// Select requires a controlling terminal even when a desktop is available. An
// unattended helper must cancel ambiguous lookups rather than choose an account.
func (d Dialog) Select(ctx context.Context, title, prompt string, choices []string) (int, error) {
	if d.Terminal == "" || len(choices) == 0 {
		return -1, ErrDenied
	}
	in, _, cleanup, err := d.openTerminal()
	if err != nil {
		return -1, ErrDenied
	}
	_ = in
	cleanup()
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	title, prompt = dialogText(title), dialogText(prompt)
	labels := make([]string, len(choices))
	for i, choice := range choices {
		labels[i] = fmt.Sprintf("%d — %s", i+1, dialogText(choice))
	}
	switch d.Backend {
	case "", "auto":
		selected, err := d.desktopSelect(ctx, title, prompt, labels)
		if !errors.Is(err, errNoDesktop) {
			return selected, err
		}
		return d.terminalSelect(ctx, title, prompt, labels)
	case "terminal":
		return d.terminalSelect(ctx, title, prompt, labels)
	case "pinentry":
		value, err := d.pinentry(ctx, title, prompt+"\n"+strings.Join(labels, "\n")+"\nEnter the option number", true)
		if err != nil {
			return -1, err
		}
		return selectionNumber(value, len(choices))
	default:
		return -1, fmt.Errorf("unknown agent dialog backend")
	}
}
func selectionNumber(value string, count int) (int, error) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 1 || number > count {
		return -1, ErrDenied
	}
	return number - 1, nil
}
func (d Dialog) terminalSelect(ctx context.Context, title, prompt string, labels []string) (int, error) {
	in, out, cleanup, err := d.openTerminal()
	if err != nil {
		return -1, err
	}
	defer cleanup()
	state, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return -1, err
	}
	defer func() { _ = term.Restore(int(in.Fd()), state) }()
	header := "\x1b[?1049h\x1b[2J\x1b[H" + title + "\r\n\r\n" + prompt + "\r\n\r\n" + strings.Join(labels, "\r\n") + "\r\n\r\nOption number (Esc to cancel): "
	if _, err := io.WriteString(out, header); err != nil {
		return -1, err
	}
	defer func() { _, _ = io.WriteString(out, "\x1b[?1049l") }()
	var value strings.Builder
	for {
		var data [4]byte
		_, err := terminalBytes(ctx, in, data[:])
		if err != nil {
			return -1, err
		}
		switch data[0] {
		case 3, 4, 27:
			return -1, ErrDenied
		case '\n', '\r':
			return selectionNumber(value.String(), len(labels))
		case 8, 127:
			text := value.String()
			if len(text) > 0 {
				value.Reset()
				value.WriteString(text[:len(text)-1])
				_, _ = io.WriteString(out, "\b \b")
			}
		default:
			if data[0] >= '0' && data[0] <= '9' && value.Len() < 9 {
				value.WriteByte(data[0])
				if _, err := out.Write(data[:1]); err != nil {
					return -1, err
				}
			}
		}
	}
}
func selectedLabel(value string, labels []string) (int, error) {
	for i, label := range labels {
		if value == label {
			return i, nil
		}
	}
	return -1, ErrDenied
}
