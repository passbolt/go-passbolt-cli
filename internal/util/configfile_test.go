package util

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/passbolt/go-passbolt-cli/internal/util/securefile"
)

// skipWithoutPOSIXModes skips a test whose whole point is a permission bit.
func skipWithoutPOSIXModes(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not POSIX on " + runtime.GOOS)
	}
}

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}

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

func TestWarnIfExposed(t *testing.T) {
	skipWithoutPOSIXModes(t)

	capture := func(t *testing.T, path string) string {
		t.Helper()
		var buf bytes.Buffer
		orig := Err
		Err = NewSanitizingWriter(&buf)
		t.Cleanup(func() { Err = orig })
		WarnIfExposed(path, ConfigExposure)
		FlushOutputs()
		return buf.String()
	}

	t.Run("warns on a shared file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "shared-data.toml")
		if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
		if got := capture(t, path); !strings.Contains(got, "mode 0644") || !strings.Contains(got, "chmod 600") {
			t.Errorf("warning = %q, want it to name mode 0644 and suggest chmod 600", got)
		}
	})

	t.Run("silent on an owner-only file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		if got := capture(t, path); got != "" {
			t.Errorf("warning = %q, want none", got)
		}
	})

	t.Run("warns once per path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "shared-data.toml")
		if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
		if got := capture(t, path); got == "" {
			t.Fatal("first call printed nothing, want a warning")
		}
		if got := capture(t, path); got != "" {
			t.Errorf("second call = %q, want nothing", got)
		}
	})

	t.Run("warns on a file another user owns", func(t *testing.T) {
		const path = "/etc/passwd"
		if os.Getuid() == 0 {
			t.Skip("running as root, which owns everything")
		}
		if _, fix, exposed := securefile.Exposure(path); !exposed || fix != "" {
			t.Skip("no foreign-owned file to check")
		}
		if got := capture(t, path); !strings.Contains(got, "not you") || strings.Count(got, "\n") != 2 {
			t.Errorf("warning = %q, want two lines saying the file is not yours", got)
		}
	})
}
