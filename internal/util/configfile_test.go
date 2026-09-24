package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestWriteViperConfig_LeavesUnsupportedFileIntact: WriteConfigTo rejects the format only
// after the caller has opened the file, so the marshal has to happen before the open.
func TestWriteViperConfig_LeavesUnsupportedFileIntact(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("serverAddress", "https://example.com")

	path := filepath.Join(t.TempDir(), "notes.txt")
	const notes = "shopping list\nmilk\n"
	if err := os.WriteFile(path, []byte(notes), 0600); err != nil {
		t.Fatal(err)
	}

	if err := WriteViperConfig(path); err == nil {
		t.Fatal("WriteViperConfig into a .txt succeeded, want an unsupported-format error")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != notes {
		t.Errorf("content = %q, want it untouched at %q", got, notes)
	}
}

func TestWriteViperConfig_WritesOwnerOnly(t *testing.T) {
	skipWithoutPOSIXModes(t)

	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("serverAddress", "https://example.com")

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := WriteViperConfig(path); err != nil {
		t.Fatalf("WriteViperConfig: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "example.com") {
		t.Errorf("content = %q, want the setting in it", got)
	}
	if got := modeOf(t, path); got != 0600 {
		t.Errorf("mode = %04o, want 0600", got)
	}
}
