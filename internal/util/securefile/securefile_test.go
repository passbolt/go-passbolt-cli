package securefile

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
	if err != nil || !info.Mode().IsRegular() || !isForeign(candidate) {
		t.Skip("no foreign-owned regular file to point a symlink at")
	}
	return candidate
}

// isForeign reports whether Exposure flags path as someone else's: exposed, with no fix.
func isForeign(path string) bool {
	_, fix, exposed := Exposure(path)
	return exposed && fix == ""
}

// TestExposure_ForeignOwner is the ownership check callers warn on.
func TestExposure_ForeignOwner(t *testing.T) {
	skipWithoutPOSIXModes(t)

	mine := filepath.Join(t.TempDir(), "mine")
	if err := os.WriteFile(mine, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if isForeign(mine) {
		t.Error("a file we just created reported as foreign")
	}
	if detail, fix, exposed := Exposure(foreignFile(t)); !exposed || fix != "" || !strings.HasPrefix(detail, "owned by ") {
		t.Errorf("Exposure on another user's file = (%q, %q, %v), want its owner and no fix", detail, fix, exposed)
	}
	if isForeign(filepath.Join(t.TempDir(), "absent")) {
		t.Error("a missing file reported as foreign")
	}
}

func TestCreate_CreatesOwnerOnly(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "config.toml")
	file, err := Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := file.WriteString("secret\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	file.Close()

	if got := modeOf(t, path); got != 0600 {
		t.Errorf("mode = %04o, want 0600", got)
	}
}

func TestCreate_TightensExistingFile(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0666); err != nil {
		t.Fatal(err)
	}

	file, err := Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	file.Close()

	if got := modeOf(t, path); got != 0600 {
		t.Errorf("mode = %04o, want 0600", got)
	}
}

func TestCreate_TruncatesExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("a very long previous config\n"), 0600); err != nil {
		t.Fatal(err)
	}

	file, err := Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
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

// TestCreate_FollowsSymlinkToOwnFile: a dotfiles config is a link like this.
func TestCreate_FollowsSymlinkToOwnFile(t *testing.T) {
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

	file, err := Create(link)
	if err != nil {
		t.Fatalf("Create: %v", err)
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

// TestCreate_FailsOnUnwritableForeignFile is the planted-link attack: ownership only warns now,
// so what stops it is that the open needs write access to the target.
func TestCreate_FailsOnUnwritableForeignFile(t *testing.T) {
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
	file, err := Create(link)
	if err == nil {
		file.Close()
		t.Fatal("Create succeeded through a foreign symlink, want error")
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("the target was modified, want it left intact")
	}
}

// TestCreate_RefusesDevice: ownership says nothing about /dev/null.
func TestCreate_RefusesDevice(t *testing.T) {
	skipWithoutPOSIXModes(t)

	link := filepath.Join(t.TempDir(), "config.toml")
	if err := os.Symlink(os.DevNull, link); err != nil {
		t.Fatal(err)
	}

	file, err := Create(link)
	if err == nil {
		file.Close()
		t.Fatal("Create succeeded onto a device, want error")
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("error = %v, want it to name the regular-file rule", err)
	}
}

func TestCreate_RefusesDirectory(t *testing.T) {
	if _, err := Create(t.TempDir()); err == nil {
		t.Fatal("Create on a directory succeeded, want error")
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

func TestExposure(t *testing.T) {
	skipWithoutPOSIXModes(t)

	t.Run("reports and changes nothing", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "shared-data.toml")
		if err := os.WriteFile(path, []byte("key = \"value\"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}

		detail, fix, exposed := Exposure(path)
		if !exposed || detail != "mode 0644" || !strings.Contains(fix, "chmod 600") {
			t.Errorf("Exposure = (%q, %q, %v), want mode 0644 and a chmod 600 fix", detail, fix, exposed)
		}
		if got := modeOf(t, path); got != 0644 {
			t.Errorf("mode = %04o, want it left at 0644", got)
		}
	})

	t.Run("not exposed: owner-only file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, exposed := Exposure(path); exposed {
			t.Error("Exposure reported an owner-only file")
		}
	})

	t.Run("not exposed: missing file", func(t *testing.T) {
		if _, _, exposed := Exposure(filepath.Join(t.TempDir(), "absent.toml")); exposed {
			t.Error("Exposure reported a missing file")
		}
	})
}
