//go:build windows

package securefile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/passbolt/go-passbolt-cli/internal/util/securefile/winacl"
)

// hostsFile is a system file this process should not own, even elevated.
const hostsFile = `C:\Windows\System32\drivers\etc\hosts`

func securityOf(t *testing.T, path string) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatalf("GetNamedSecurityInfo %s: %v", path, err)
	}
	return sd
}

// setDACL replaces path's DACL from SDDL, e.g. "D:(A;;FR;;;WD)" to let Everyone read it.
func setDACL(t *testing.T, path, sddl string) {
	t.Helper()
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		t.Fatalf("SecurityDescriptorFromString %q: %v", sddl, err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	info := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION)
	if strings.HasPrefix(sddl, "D:P") {
		info |= windows.PROTECTED_DACL_SECURITY_INFORMATION
	} else {
		info |= windows.UNPROTECTED_DACL_SECURITY_INFORMATION
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, info, nil, nil, dacl, nil); err != nil {
		t.Fatalf("SetNamedSecurityInfo %s: %v", path, err)
	}
}

func callerSID(t *testing.T) *windows.SID {
	t.Helper()
	user, _, err := winacl.CallerSIDs()
	if err != nil {
		t.Fatal(err)
	}
	return user
}

// assertPrivate checks path has a protected DACL of exactly winacl.PrivateSD's three entries.
func assertPrivate(t *testing.T, path string, dir bool) {
	t.Helper()
	sd := securityOf(t, path)
	control, _, err := sd.Control()
	if err != nil {
		t.Fatal(err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Errorf("%s: DACL not protected, it still inherits: %s", path, sd)
	}
	if sid, loose := winacl.LooseTrustee(sd); loose {
		t.Errorf("%s: DACL exposes it to %v: %s", path, sid, sd)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if dacl.AceCount != 3 {
		t.Errorf("%s: %d ACEs, want 3: %s", path, dacl.AceCount, sd)
	}
	if dir && !strings.Contains(sd.String(), "OICI") {
		t.Errorf("%s: directory ACEs do not propagate to children: %s", path, sd)
	}
}

func TestOwnedByCaller_Windows(t *testing.T) {
	mine := filepath.Join(t.TempDir(), "mine")
	if err := os.WriteFile(mine, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if !ownedByCaller(openForStat(t, mine)) {
		t.Error("ownedByCaller on a file we just created = false, want true")
	}
	if ownedByCaller(openForStat(t, hostsFile)) {
		t.Errorf("ownedByCaller on %s = true, want false", hostsFile)
	}
}

func TestCreate_Windows_CreatesPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	file, err := Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	file.Close()
	assertPrivate(t, path, false)
}

// TestCreate_Windows_TightensInheritedAccess is a --config under a shared directory.
func TestCreate_Windows_TightensInheritedAccess(t *testing.T) {
	shared := t.TempDir()
	setDACL(t, shared, fmt.Sprintf("D:P(A;OICI;FA;;;%s)(A;OICI;FR;;;BU)", callerSID(t)))
	path := filepath.Join(shared, "config.toml")
	if err := os.WriteFile(path, []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, loose := winacl.LooseTrustee(securityOf(t, path)); !loose {
		t.Fatal("setup: the file did not inherit BUILTIN\\Users read")
	}

	file, err := Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	file.Close()
	assertPrivate(t, path, false)
}

func TestCreate_Windows_TightensExplicitGrant(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	setDACL(t, path, fmt.Sprintf("D:(A;;FA;;;%s)(A;;FR;;;WD)", callerSID(t)))

	file, err := Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	file.Close()
	assertPrivate(t, path, false)
}

// TestCreate_Windows_RefusesForeignFile covers both a direct path and a planted link.
func TestCreate_Windows_RefusesForeignFile(t *testing.T) {
	before, err := os.ReadFile(hostsFile)
	if err != nil {
		t.Skipf("reading %s: %v", hostsFile, err)
	}
	if ownedByCaller(openForStat(t, hostsFile)) {
		t.Skipf("this process owns %s", hostsFile)
	}
	targets := []string{hostsFile}
	link := filepath.Join(t.TempDir(), "config.toml")
	if err := os.Symlink(hostsFile, link); err == nil {
		targets = append(targets, link)
	} else {
		t.Logf("no symlink privilege, testing the direct path only: %v", err)
	}

	for _, target := range targets {
		file, err := Create(target)
		if err == nil {
			file.Close()
			t.Errorf("Create(%s) succeeded onto a file we do not own, want error", target)
		}
	}
	after, err := os.ReadFile(hostsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("%s was modified, want it left intact", hostsFile)
	}
}

func TestCreate_Windows_RefusesDevice(t *testing.T) {
	file, err := Create("NUL")
	if err == nil {
		file.Close()
		t.Fatal("Create(NUL) succeeded, want error")
	}
}

func TestRepairMode_Windows(t *testing.T) {
	t.Run("tightens an inherited directory", func(t *testing.T) {
		shared := t.TempDir()
		setDACL(t, shared, fmt.Sprintf("D:P(A;OICI;FA;;;%s)(A;OICI;FR;;;BU)", callerSID(t)))
		dir := filepath.Join(shared, "go-passbolt-cli")
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := RepairMode(dir, 0700); err != nil {
			t.Fatalf("RepairMode: %v", err)
		}
		assertPrivate(t, dir, true)

		// A file created afterwards inherits the private ACL.
		child := filepath.Join(dir, "go-passbolt-cli.toml")
		if err := os.WriteFile(child, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		if sid, loose := winacl.LooseTrustee(securityOf(t, child)); loose {
			t.Errorf("new file in the repaired directory is exposed to %v", sid)
		}
	})

	t.Run("tightens a file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		setDACL(t, path, fmt.Sprintf("D:(A;;FA;;;%s)(A;;FR;;;WD)", callerSID(t)))
		if err := RepairMode(path, 0600); err != nil {
			t.Fatalf("RepairMode: %v", err)
		}
		assertPrivate(t, path, false)
	})

	t.Run("leaves a private file alone", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		before := securityOf(t, path).String()
		if err := RepairMode(path, 0600); err != nil {
			t.Fatalf("RepairMode: %v", err)
		}
		if after := securityOf(t, path).String(); after != before {
			t.Errorf("DACL changed from %s to %s, want it untouched", before, after)
		}
	})

	t.Run("missing path is not an error", func(t *testing.T) {
		if err := RepairMode(filepath.Join(t.TempDir(), "absent.toml"), 0600); err != nil {
			t.Errorf("RepairMode on a missing file = %v, want nil", err)
		}
	})
}

func TestExposure_Windows(t *testing.T) {
	t.Run("reports and changes nothing", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "shared-data.toml")
		if err := os.WriteFile(path, []byte("key = \"value\"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		setDACL(t, path, fmt.Sprintf("D:(A;;FA;;;%s)(A;;FR;;;WD)", callerSID(t)))
		before := securityOf(t, path).String()

		everyone, err := windows.CreateWellKnownSid(windows.WinWorldSid)
		if err != nil {
			t.Fatal(err)
		}
		// Looked up rather than spelled out: account names are localized.
		name := winacl.AccountName(everyone)
		detail, fix, exposed := Exposure(path)
		if !exposed || !strings.Contains(detail, name) || !strings.Contains(fix, "icacls") {
			t.Errorf("Exposure = (%q, %q, %v), want it to name %s and suggest icacls", detail, fix, exposed, name)
		}
		if after := securityOf(t, path).String(); after != before {
			t.Errorf("DACL changed from %s to %s, want it untouched", before, after)
		}
	})

	t.Run("not exposed: private file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		file, err := Create(path)
		if err != nil {
			t.Fatal(err)
		}
		file.Close()
		if _, _, exposed := Exposure(path); exposed {
			t.Error("Exposure reported a private file")
		}
	})

	t.Run("not exposed: missing file", func(t *testing.T) {
		if _, _, exposed := Exposure(filepath.Join(t.TempDir(), "absent.toml")); exposed {
			t.Error("Exposure reported a missing file")
		}
	})
}
