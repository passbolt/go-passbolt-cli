//go:build windows

package util

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// hasPOSIXModes is false: an ACL, not the mode bits, decides who can read a file here.
const hasPOSIXModes = false

// exposingRights are the rights that let another principal read, change or re-permission a file.
const exposingRights = windows.FILE_READ_DATA | windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA |
	windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER |
	windows.GENERIC_READ | windows.GENERIC_WRITE | windows.GENERIC_ALL

// callerSIDs returns the token's user and the owners it can stamp on a new file: the user, and
// Administrators when elevated. Other owner groups a token may carry are not trusted.
func callerSIDs() (user *windows.SID, owners []*windows.SID, err error) {
	token := windows.GetCurrentProcessToken()
	tu, err := token.GetTokenUser()
	if err != nil {
		return nil, nil, err
	}
	user = tu.User.Sid
	owners = []*windows.SID{user}

	groups, err := token.GetTokenGroups()
	if err != nil {
		return nil, nil, err
	}
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, nil, err
	}
	const usable = windows.SE_GROUP_OWNER | windows.SE_GROUP_ENABLED
	for _, g := range groups.AllGroups() {
		// A filtered (unelevated) token keeps Administrators, but deny-only and without SE_GROUP_OWNER.
		if g.Attributes&usable == usable && g.Attributes&windows.SE_GROUP_USE_FOR_DENY_ONLY == 0 && g.Sid.Equals(admins) {
			owners = append(owners, admins)
			break
		}
	}
	return user, owners, nil
}

// privateSD is the protected descriptor that stands in for 0600, or 0700 with inheritance for a directory.
func privateSD(dir bool) (*windows.SECURITY_DESCRIPTOR, error) {
	user, _, err := callerSIDs()
	if err != nil {
		return nil, err
	}
	inherit := ""
	if dir {
		inherit = "OICI"
	}
	return windows.SecurityDescriptorFromString(fmt.Sprintf("D:P(A;%[1]s;FA;;;%[2]s)(A;%[1]s;FA;;;SY)(A;%[1]s;FA;;;BA)", inherit, user))
}

// openSecret opens path for writing, creating it with privateSD; WRITE_DAC lets tightenIfLoose fix it.
func openSecret(path string) (*os.File, error) {
	sd, err := privateSD(false)
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

// ownedByCaller reports whether the open file's owner SID is one callerSIDs says the caller can stamp.
func ownedByCaller(file *os.File) bool {
	sd, err := windows.GetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return false
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil {
		return false
	}
	_, owners, err := callerSIDs()
	return err == nil && containsSID(owners, owner)
}

// looseTrustee returns whom sd's DACL exposes the object to; the SID is nil when that is not one principal.
// It reads the DACL's SDDL rendering, since walking the ACEs in memory takes unsafe pointer casts,
// and fails closed on anything it cannot parse. sd must hold only a DACL (DACL_SECURITY_INFORMATION).
func looseTrustee(sd *windows.SECURITY_DESCRIPTOR) (*windows.SID, bool) {
	// String is "" on failure, which does not parse. No DACL, or a NULL one, grants everyone everything.
	aces, err := parseSDDLDACL(sd.String())
	if err != nil {
		return nil, true
	}
	trusted, err := trustedSIDs()
	if err != nil {
		return nil, true
	}
	for _, ace := range aces {
		if ace.inheritOnly() {
			continue // applies to children only
		}
		switch ace.Type {
		case "A":
		case "D", "OD", "XD":
			continue // deny entries take nothing away from the owner
		default:
			return nil, true // object, callback and any other grant whose scope this does not evaluate
		}
		if ace.Mask&exposingRights == 0 {
			continue
		}
		sid, err := windows.StringToSid(ace.SID)
		if err != nil {
			return nil, true
		}
		if !containsSID(trusted, sid) {
			return sid, true
		}
	}
	return nil, false
}

// trustedSIDs are the principals privateSD grants, plus OWNER RIGHTS.
func trustedSIDs() ([]*windows.SID, error) {
	user, _, err := callerSIDs()
	if err != nil {
		return nil, err
	}
	trusted := []*windows.SID{user}
	for _, k := range []windows.WELL_KNOWN_SID_TYPE{windows.WinLocalSystemSid, windows.WinBuiltinAdministratorsSid, windows.WinCreatorOwnerRightsSid} {
		sid, err := windows.CreateWellKnownSid(k)
		if err != nil {
			return nil, err
		}
		trusted = append(trusted, sid)
	}
	return trusted, nil
}

func containsSID(sids []*windows.SID, sid *windows.SID) bool {
	for _, s := range sids {
		if s.Equals(sid) {
			return true
		}
	}
	return false
}

// tightenIfLoose replaces a DACL that exposes the file with privateSD's; CreateFile sets it only on creation.
func tightenIfLoose(file *os.File, info fs.FileInfo) error {
	h := windows.Handle(file.Fd())
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	if _, loose := looseTrustee(sd); !loose {
		return nil
	}
	private, err := privateSD(info.IsDir())
	if err != nil {
		return err
	}
	dacl, _, err := private.DACL()
	if err != nil {
		return err
	}
	return windows.SetSecurityInfo(h, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
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
	sid, loose := looseTrustee(sd)
	if !loose {
		return "", false
	}
	if sid == nil {
		return "its ACL grants access beyond you, SYSTEM and Administrators", true
	}
	return "its ACL grants access to " + accountName(sid), true
}

func accountName(sid *windows.SID) string {
	account, domain, _, err := sid.LookupAccount("")
	if err != nil {
		return sid.String()
	}
	if domain == "" {
		return account
	}
	return domain + `\` + account
}

// fixCommand drops inherited entries and grants the caller full control; explicit grants to others still need /remove.
func fixCommand(path string) string {
	who := "%USERNAME%"
	if user, _, err := callerSIDs(); err == nil {
		who = accountName(user)
	}
	return fmt.Sprintf(`icacls "%s" /inheritance:r /grant:r "%s:F"`, path, who)
}
