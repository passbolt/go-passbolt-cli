//go:build !windows

package util

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

func TestCreateSecretFile_RefusesWhenChmodIgnored(t *testing.T) {
	ignoreChmod(t)
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	file, err := CreateSecretFile(path)
	if err == nil {
		file.Close()
		t.Fatal("CreateSecretFile wrote a file that is still 0644, want error")
	}
	if !errors.Is(err, errPermissionsIgnored) {
		t.Errorf("err = %v, want errPermissionsIgnored", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old\n" {
		t.Errorf("content = %q, want it untouched", got)
	}
	if mode := modeOf(t, path); mode != 0644 {
		t.Errorf("mode = %04o, want 0644", mode)
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
