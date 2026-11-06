//go:build !windows

package securefile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// ignoreChmod plays a mount such as vfat with "quiet", where chmod succeeds and changes nothing.
func ignoreChmod(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { fchmod, chmodPath = (*os.File).Chmod, os.Chmod })
	fchmod = func(*os.File, os.FileMode) error { return nil }
	chmodPath = func(string, os.FileMode) error { return nil }
}

// TestCreate_WritesWhenChmodIgnored: a disk that keeps no permissions is the user's choice, so
// Create writes anyway and Exposure still reports it for the caller's warning.
func TestCreate_WritesWhenChmodIgnored(t *testing.T) {
	ignoreChmod(t)
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("a longer old config\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	file, err := Create(path)
	if err != nil {
		t.Fatalf("Create: %v, want it to write despite the loose mode", err)
	}
	if _, err := file.WriteString("new\n"); err != nil {
		t.Fatal(err)
	}
	file.Close()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Errorf("content = %q, want %q", got, "new\n")
	}
	if mode := modeOf(t, path); mode != 0644 {
		t.Errorf("mode = %04o, want 0644", mode)
	}
	if _, _, exposed := Exposure(path); !exposed {
		t.Error("Exposure did not report the loose file, so nobody would be warned")
	}
}

func TestRepairMode_ReportsIgnoredChmod(t *testing.T) {
	ignoreChmod(t)
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if err := RepairMode(path, 0600); !errors.Is(err, errPermissionsIgnored) {
		t.Errorf("RepairMode = %v, want errPermissionsIgnored", err)
	}
}
