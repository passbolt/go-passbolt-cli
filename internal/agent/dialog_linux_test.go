package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopHelperSelectionAndDenial(t *testing.T) {
	for _, helper := range []string{"zenity", "kdialog"} {
		t.Run(helper, func(t *testing.T) {
			dir := t.TempDir()
			log := filepath.Join(dir, "arguments")
			script := `#!/bin/sh
printf '%s\n' "$@" > "$DIALOG_TEST_LOG"
if [ "$DIALOG_TEST_DENY" = 1 ]; then exit 1; fi
printf 'fixture-value\n'
`
			if err := os.WriteFile(filepath.Join(dir, helper), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			t.Setenv("DIALOG_TEST_LOG", log)
			t.Setenv("DISPLAY", ":test")
			dialog := Dialog{Backend: "auto", Environment: os.Environ()}
			value, err := dialog.Ask(context.Background(), "title", "prompt", true)
			if err != nil || value != "fixture-value" {
				t.Fatalf("secret input: %q %v", value, err)
			}
			args, _ := os.ReadFile(log)
			expected := "--password"
			if helper == "zenity" {
				expected = "--hide-text"
			}
			if !strings.Contains(string(args), expected) {
				t.Fatalf("secret entry was not masked: %s", args)
			}
			t.Setenv("DIALOG_TEST_DENY", "1")
			dialog.Environment = os.Environ()
			err = dialog.Confirm(context.Background(), "title", "prompt")
			if !errors.Is(err, ErrDenied) {
				t.Fatalf("explicit denial fell through to another backend: %v", err)
			}
			args, _ = os.ReadFile(log)
			expected = "--defaultno"
			if helper == "zenity" {
				expected = "--default-cancel"
			}
			if !strings.Contains(string(args), expected) {
				t.Fatalf("approval did not default to denial: %s", args)
			}
		})
	}
}
func TestDesktopRequiresDisplay(t *testing.T) {
	_, err := (Dialog{Environment: []string{"DISPLAY=", "WAYLAND_DISPLAY="}}).desktop(context.Background(), "title", "prompt", false)
	if !errors.Is(err, errNoDesktop) {
		t.Fatalf("desktop selected without a display: %v", err)
	}
}

func TestResourceApprovalRendersFullUnicodeNameAndRequester(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "arguments")
	if err := os.WriteFile(filepath.Join(dir, "zenity"), []byte(`#!/bin/sh
printf '%s\n' "$@" > "$DIALOG_TEST_LOG"
`), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("DIALOG_TEST_LOG", log)
	t.Setenv("DISPLAY", ":test")
	name := strings.Repeat("🌐", 255) + "resource-tail"
	title, prompt := ResourceApproval("vault", "get resource", "git (PID 123)", "resource-uuid", name)
	if err := (Dialog{Environment: os.Environ()}).Confirm(context.Background(), title, prompt); err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{name, "resource-uuid", "git (PID 123)"} {
		if !strings.Contains(string(args), want) {
			t.Fatalf("rendered approval missing %q", want)
		}
	}
}
