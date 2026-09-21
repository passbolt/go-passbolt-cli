package util

import (
	"bufio"
	"errors"
	"os"
	"strings"

	"golang.org/x/term"
)

// ErrNotInteractive is returned by Confirm when there is no terminal to ask.
var ErrNotInteractive = errors.New("stdin is not a terminal")

// IsInteractive reports whether there is a user at the other end of stdin.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// Confirm asks a yes/no question on stderr; a non-terminal gets ErrNotInteractive.
func Confirm(prompt string) (bool, error) {
	if !IsInteractive() {
		return false, ErrNotInteractive
	}

	Eprint(prompt)
	FlushOutputs()

	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
