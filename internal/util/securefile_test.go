package util

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// skipWithoutPOSIXModes skips a test whose whole point is a permission bit.
func skipWithoutPOSIXModes(t *testing.T) {
	t.Helper()
	if !hasPOSIXModes {
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

// foreignFile names a regular file this process does not own, or skips the test.
func foreignFile(t *testing.T) string {
	t.Helper()
	if os.Getuid() == 0 {
		t.Skip("running as root, which owns everything")
	}
	const candidate = "/etc/passwd"
	info, err := os.Stat(candidate)
	if err != nil || !info.Mode().IsRegular() || ownedByCaller(openForStat(t, candidate)) {
		t.Skip("no foreign-owned regular file to point a symlink at")
	}
	return candidate
}

// openForStat opens path read-only for the ownership check, closing it with the test.
func openForStat(t *testing.T, path string) *os.File {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { file.Close() })
	return file
}

// TestOwnedByCaller is the ownership predicate CreateSecretFile refuses on.
func TestOwnedByCaller(t *testing.T) {
	skipWithoutPOSIXModes(t)

	mine := filepath.Join(t.TempDir(), "mine")
	if err := os.WriteFile(mine, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if !ownedByCaller(openForStat(t, mine)) {
		t.Error("ownedByCaller on a file we just created = false, want true")
	}

	if ownedByCaller(openForStat(t, foreignFile(t))) {
		t.Error("ownedByCaller on another user's file = true, want false")
	}
}

func TestCreateSecretFile_CreatesOwnerOnly(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "config.toml")
	file, err := CreateSecretFile(path)
	if err != nil {
		t.Fatalf("CreateSecretFile: %v", err)
	}
	if _, err := file.WriteString("secret\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	file.Close()

	if got := modeOf(t, path); got != 0600 {
		t.Errorf("mode = %04o, want 0600", got)
	}
}

func TestCreateSecretFile_TightensExistingFile(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0666); err != nil {
		t.Fatal(err)
	}

	file, err := CreateSecretFile(path)
	if err != nil {
		t.Fatalf("CreateSecretFile: %v", err)
	}
	file.Close()

	if got := modeOf(t, path); got != 0600 {
		t.Errorf("mode = %04o, want 0600", got)
	}
}

func TestCreateSecretFile_TruncatesExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("a very long previous config\n"), 0600); err != nil {
		t.Fatal(err)
	}

	file, err := CreateSecretFile(path)
	if err != nil {
		t.Fatalf("CreateSecretFile: %v", err)
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
}

// TestCreateSecretFile_FollowsSymlinkToOwnFile: a dotfiles config is a link like this.
func TestCreateSecretFile_FollowsSymlinkToOwnFile(t *testing.T) {
	skipWithoutPOSIXModes(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles.toml")
	if err := os.WriteFile(target, []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.toml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	file, err := CreateSecretFile(link)
	if err != nil {
		t.Fatalf("CreateSecretFile: %v", err)
	}
	if _, err := file.WriteString("new\n"); err != nil {
		t.Fatal(err)
	}
	file.Close()

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Errorf("target content = %q, want %q", got, "new\n")
	}
	if got := modeOf(t, target); got != 0600 {
		t.Errorf("target mode = %04o, want 0600", got)
	}
	// The link itself stays a link, rather than being replaced by a regular file.
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link mode = %v, want it still a symlink", info.Mode())
	}
}

// TestCreateSecretFile_RefusesSymlinkToAnotherUsersFile is the planted-link attack.
func TestCreateSecretFile_RefusesSymlinkToAnotherUsersFile(t *testing.T) {
	skipWithoutPOSIXModes(t)

	target := foreignFile(t)
	link := filepath.Join(t.TempDir(), "config.toml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	file, err := CreateSecretFile(link)
	if err == nil {
		file.Close()
		t.Fatal("CreateSecretFile succeeded through a foreign symlink, want error")
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("the target was modified, want it left intact")
	}
}

// TestCreateSecretFile_RefusesDevice: ownership says nothing about /dev/null.
func TestCreateSecretFile_RefusesDevice(t *testing.T) {
	skipWithoutPOSIXModes(t)

	link := filepath.Join(t.TempDir(), "config.toml")
	if err := os.Symlink(os.DevNull, link); err != nil {
		t.Fatal(err)
	}

	file, err := CreateSecretFile(link)
	if err == nil {
		file.Close()
		t.Fatal("CreateSecretFile succeeded onto a device, want error")
	}
	if !strings.Contains(err.Error(), "not a regular file you own") {
		t.Errorf("error = %v, want it to name the regular-file rule", err)
	}
}

func TestCreateSecretFile_RefusesDirectory(t *testing.T) {
	if _, err := CreateSecretFile(t.TempDir()); err == nil {
		t.Fatal("CreateSecretFile on a directory succeeded, want error")
	}
}

func TestRepairMode(t *testing.T) {
	skipWithoutPOSIXModes(t)

	t.Run("tightens a loose file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
		if err := RepairMode(path, 0600); err != nil {
			t.Fatalf("RepairMode: %v", err)
		}
		if got := modeOf(t, path); got != 0600 {
			t.Errorf("mode = %04o, want 0600", got)
		}
	})

	t.Run("tightens a loose directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "go-passbolt-cli")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := RepairMode(dir, 0700); err != nil {
			t.Fatalf("RepairMode: %v", err)
		}
		if got := modeOf(t, dir); got != 0700 {
			t.Errorf("mode = %04o, want 0700", got)
		}
	})

	t.Run("missing path is not an error", func(t *testing.T) {
		if err := RepairMode(filepath.Join(t.TempDir(), "absent.toml"), 0600); err != nil {
			t.Errorf("RepairMode on a missing file = %v, want nil", err)
		}
	})
}

func TestWarnIfGroupOrWorldAccessible(t *testing.T) {
	skipWithoutPOSIXModes(t)

	capture := func(t *testing.T, path string) string {
		t.Helper()
		var buf bytes.Buffer
		orig := Err
		Err = NewSanitizingWriter(&buf)
		t.Cleanup(func() { Err = orig })
		WarnIfGroupOrWorldAccessible(path)
		FlushOutputs()
		return buf.String()
	}

	t.Run("warns and changes nothing", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "shared-data.toml")
		if err := os.WriteFile(path, []byte("key = \"value\"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}

		got := capture(t, path)
		if !strings.Contains(got, "chmod 600") {
			t.Errorf("warning = %q, want it to suggest chmod 600", got)
		}
		if got := modeOf(t, path); got != 0644 {
			t.Errorf("mode = %04o, want it left at 0644", got)
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

	t.Run("silent on a missing file", func(t *testing.T) {
		if got := capture(t, filepath.Join(t.TempDir(), "absent.toml")); got != "" {
			t.Errorf("warning = %q, want none", got)
		}
	})
}
