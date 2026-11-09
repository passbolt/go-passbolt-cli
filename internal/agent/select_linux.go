package agent

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

func (d Dialog) desktopSelect(ctx context.Context, title, prompt string, labels []string) (int, error) {
	display := false
	for _, entry := range d.Environment {
		if (strings.HasPrefix(entry, "DISPLAY=") && entry != "DISPLAY=") || (strings.HasPrefix(entry, "WAYLAND_DISPLAY=") && entry != "WAYLAND_DISPLAY=") {
			display = true
		}
	}
	if !display {
		return -1, errNoDesktop
	}
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--list", "--title", title, "--text", prompt, "--column", "Resource", "--hide-header", "--"}
		args = append(args, labels...)
		value, err := d.run(ctx, "zenity", args, "")
		if err != nil {
			return -1, err
		}
		return selectedLabel(value, labels)
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		args := []string{"--title", title, "--menu", prompt}
		for i, label := range labels {
			args = append(args, strconv.Itoa(i+1), label)
		}
		value, err := d.run(ctx, "kdialog", args, "")
		if err != nil {
			return -1, err
		}
		return selectionNumber(value, len(labels))
	}
	return -1, errNoDesktop
}
