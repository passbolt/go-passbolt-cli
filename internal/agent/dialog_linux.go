package agent

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

func (d Dialog) desktop(ctx context.Context, title, prompt string, secret bool) (string, error) {
	display := false
	for _, entry := range d.Environment {
		if (strings.HasPrefix(entry, "DISPLAY=") && entry != "DISPLAY=") || (strings.HasPrefix(entry, "WAYLAND_DISPLAY=") && entry != "WAYLAND_DISPLAY=") {
			display = true
		}
	}
	if !display {
		return "", errNoDesktop
	}
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--title", title, "--text", prompt}
		if secret {
			args = append(args, "--entry", "--hide-text")
		} else {
			args = append(args, "--question", "--default-cancel", "--ok-label=Allow", "--cancel-label=Deny")
		}
		value, err := d.run(ctx, "zenity", args, "")
		if err != nil && !errors.Is(err, ErrDenied) && ctx.Err() == nil {
			return "", errNoDesktop
		}
		return value, err
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		args := []string{"--title", title}
		if secret {
			args = append(args, "--password", prompt)
		} else {
			args = append(args, "--defaultno", "--yesno", prompt)
		}
		value, err := d.run(ctx, "kdialog", args, "")
		if err != nil && !errors.Is(err, ErrDenied) && ctx.Err() == nil {
			return "", errNoDesktop
		}
		return value, err
	}
	return "", errNoDesktop
}
