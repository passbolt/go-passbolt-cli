//go:build windows

package winacl

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// ExposingRights are the rights that let another principal read, change or re-permission a file.
const ExposingRights = windows.FILE_READ_DATA | windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA |
	windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER |
	windows.GENERIC_READ | windows.GENERIC_WRITE | windows.GENERIC_ALL

// CallerSIDs returns the token's user and the owners it can stamp on a new file: the user, and
// Administrators when elevated. Other owner groups a token may carry are not trusted.
func CallerSIDs() (user *windows.SID, owners []*windows.SID, err error) {
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

// PrivateSD is the protected descriptor that stands in for 0600, or 0700 with inheritance for a directory.
func PrivateSD(dir bool) (*windows.SECURITY_DESCRIPTOR, error) {
	user, _, err := CallerSIDs()
	if err != nil {
		return nil, err
	}
	inherit := ""
	if dir {
		inherit = "OICI"
	}
	return windows.SecurityDescriptorFromString(fmt.Sprintf("D:P(A;%[1]s;FA;;;%[2]s)(A;%[1]s;FA;;;SY)(A;%[1]s;FA;;;BA)", inherit, user))
}

// LooseTrustee returns whom sd's DACL exposes the object to; the SID is nil when that is not one principal.
// It reads the DACL's SDDL rendering, since walking the ACEs in memory takes unsafe pointer casts,
// and fails closed on anything it cannot parse. sd must hold only a DACL (DACL_SECURITY_INFORMATION).
func LooseTrustee(sd *windows.SECURITY_DESCRIPTOR) (*windows.SID, bool) {
	// String is "" on failure, which does not parse. No DACL, or a NULL one, grants everyone everything.
	aces, err := ParseDACL(sd.String())
	if err != nil {
		return nil, true
	}
	trusted, err := TrustedSIDs()
	if err != nil {
		return nil, true
	}
	for _, ace := range aces {
		if ace.InheritOnly() {
			continue // applies to children only
		}
		switch ace.Type {
		case "A":
		case "D", "OD", "XD":
			continue // deny entries take nothing away from the owner
		default:
			return nil, true // object, callback and any other grant whose scope this does not evaluate
		}
		if ace.Mask&ExposingRights == 0 {
			continue
		}
		sid, err := windows.StringToSid(ace.SID)
		if err != nil {
			return nil, true
		}
		if !ContainsSID(trusted, sid) {
			return sid, true
		}
	}
	return nil, false
}

// TrustedSIDs are the principals PrivateSD grants, plus OWNER RIGHTS.
func TrustedSIDs() ([]*windows.SID, error) {
	user, _, err := CallerSIDs()
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

// ContainsSID reports whether sid is one of sids.
func ContainsSID(sids []*windows.SID, sid *windows.SID) bool {
	for _, s := range sids {
		if s.Equals(sid) {
			return true
		}
	}
	return false
}

// AccountName renders sid as DOMAIN\account, or as its S-1-... string when it does not resolve.
func AccountName(sid *windows.SID) string {
	account, domain, _, err := sid.LookupAccount("")
	if err != nil {
		return sid.String()
	}
	if domain == "" {
		return account
	}
	return domain + `\` + account
}
