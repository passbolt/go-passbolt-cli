//go:build windows

package securefile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/passbolt/go-passbolt-cli/internal/util/securefile/winacl"
)

// hasPOSIXModes is false: an ACL, not the mode bits, decides who can read a file here.
const hasPOSIXModes = false

// openSecret opens path for writing, creating it with winacl.PrivateSD; WRITE_DAC lets tightenIfLoose fix it.
func openSecret(path string) (*os.File, error) {
	sd, err := winacl.PrivateSD(false)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: path, Err: err}
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: path, Err: err}
	}
	sa := &windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	h, err := windows.CreateFile(name, windows.GENERIC_WRITE|windows.READ_CONTROL|windows.WRITE_DAC,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, sa, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}

// ownedByCaller reports whether the open file's owner SID is one winacl.CallerSIDs says the caller can stamp.
func ownedByCaller(file *os.File) bool {
	sd, err := windows.GetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return false
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil {
		return false
	}
	_, owners, err := winacl.CallerSIDs()
	return err == nil && winacl.ContainsSID(owners, owner)
}

// tightenIfLoose replaces a DACL that exposes the file with winacl.PrivateSD's; CreateFile sets it only on creation.
func tightenIfLoose(file *os.File, info fs.FileInfo) error {
	h := windows.Handle(file.Fd())
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	if _, loose := winacl.LooseTrustee(sd); !loose {
		return nil
	}
	private, err := winacl.PrivateSD(info.IsDir())
	if err != nil {
		return err
	}
	dacl, _, err := private.DACL()
	if err != nil {
		return err
	}
	if err := windows.SetSecurityInfo(h, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		return err
	}
	// Read back: a drive can accept the new DACL and keep reporting the old one.
	if sd, err = windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION); err != nil {
		return err
	}
	if _, loose := winacl.LooseTrustee(sd); loose {
		return errPermissionsIgnored
	}
	return nil
}

// repairMode ignores mode: the file or directory ACL follows from what path turns out to be.
func repairMode(path string, _ os.FileMode) error {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	// CreateFile is Windows' open call and refuses directories unless passed FILE_FLAG_BACKUP_SEMANTICS; symlinks are followed, like os.Stat.
	h, err := windows.CreateFile(
		name,
		windows.READ_CONTROL|windows.WRITE_DAC|windows.FILE_READ_ATTRIBUTES, // FILE_READ_ATTRIBUTES for Stat
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return &fs.PathError{Op: "open", Path: path, Err: err} // access denied if the target is not ours
	}
	file := os.NewFile(uintptr(h), path)
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	return tightenIfLoose(file, info)
}

// describeLoose reports whether path's ACL exposes it, naming the principal when there is one.
func describeLoose(path string) (string, bool) {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return "", false
	}
	sid, loose := winacl.LooseTrustee(sd)
	if !loose {
		return "", false
	}
	if sid == nil {
		return "its ACL grants access beyond you, SYSTEM and Administrators", true
	}
	return "its ACL grants access to " + winacl.AccountName(sid), true
}

// fixCommand drops inherited entries and grants the caller full control; explicit grants to others still need /remove.
func fixCommand(path string) string {
	who := "%USERNAME%"
	if user, _, err := winacl.CallerSIDs(); err == nil {
		who = winacl.AccountName(user)
	}
	return fmt.Sprintf(`icacls "%s" /inheritance:r /grant:r "%s:F"`, path, who)
}
