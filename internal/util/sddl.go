package util

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// This file has no build tag so its parser is tested on every platform; only Windows calls it.

// sddlACE is one ACE of a DACL in SDDL, as ConvertSecurityDescriptorToStringSecurityDescriptor writes it.
type sddlACE struct {
	Type  string // "A" allow, "D" deny, "OA" object allow, ...
	Flags string // two-letter codes: "OI", "CI", "IO", "ID", ...
	Mask  uint32
	SID   string // "S-1-5-..." or an alias such as "BA", for windows.StringToSid
}

// inheritOnly reports whether the ACE applies to children only.
func (a sddlACE) inheritOnly() bool {
	for i := 0; i+1 < len(a.Flags); i += 2 {
		if a.Flags[i:i+2] == "IO" {
			return true
		}
	}
	return false
}

// sddlRights are the SDDL access right codes, from sddl.h.
var sddlRights = map[string]uint32{
	"GA": 0x10000000, "GR": 0x80000000, "GW": 0x40000000, "GX": 0x20000000,
	"RC": 0x00020000, "SD": 0x00010000, "WD": 0x00040000, "WO": 0x00080000,
	"RP": 0x10, "WP": 0x20, "CC": 0x01, "DC": 0x02, "LC": 0x04, "SW": 0x08, "LO": 0x80, "DT": 0x40, "CR": 0x100,
	"FA": 0x001F01FF, "FR": 0x00120089, "FW": 0x00120116, "FX": 0x001200A0,
	"KA": 0x000F003F, "KR": 0x00020019, "KW": 0x00020006, "KX": 0x00020019,
	"NR": 0x2, "NW": 0x1, "NX": 0x4,
}

var errNoDACL = errors.New("no DACL")

// parseSDDLDACL returns the entries of a descriptor that holds only a DACL, rendered as
// "D:<flags>(ace)(ace)...". A NULL DACL, which grants everyone everything, is errNoDACL.
// Anything else it does not expect is an error too, so the caller fails closed: other
// sections, and conditional ACEs, whose expressions it does not read.
func parseSDDLDACL(sddl string) ([]sddlACE, error) {
	body, ok := strings.CutPrefix(sddl, "D:")
	if !ok {
		return nil, fmt.Errorf("%q is not a DACL alone", sddl)
	}
	flags, aces, found := strings.Cut(body, "(")
	if strings.Contains(flags, "NO_ACCESS_CONTROL") {
		return nil, errNoDACL
	}
	if strings.Trim(flags, "PAIR") != "" {
		return nil, fmt.Errorf("unexpected DACL flags %q", flags)
	}
	if !found {
		return nil, nil // an empty DACL
	}
	aces, ok = strings.CutSuffix(aces, ")")
	if !ok {
		return nil, fmt.Errorf("unterminated ACE list %q", sddl)
	}
	var out []sddlACE
	for s := range strings.SplitSeq(aces, ")(") {
		ace, err := parseSDDLACE(s)
		if err != nil {
			return nil, err
		}
		out = append(out, ace)
	}
	return out, nil
}

// parseSDDLACE parses "type;flags;rights;object;inherited object;sid".
func parseSDDLACE(s string) (sddlACE, error) {
	if strings.ContainsAny(s, `()"`) {
		return sddlACE{}, fmt.Errorf("ACE %q is conditional or malformed", s)
	}
	f := strings.Split(s, ";")
	if len(f) != 6 {
		return sddlACE{}, fmt.Errorf("ACE %q has %d fields, want 6", s, len(f))
	}
	ace := sddlACE{Type: f[0], Flags: f[1], SID: f[5]}
	if ace.Type == "" || ace.SID == "" {
		return sddlACE{}, fmt.Errorf("ACE %q lacks a type or a SID", s)
	}
	if len(ace.Flags)%2 != 0 {
		return sddlACE{}, fmt.Errorf("ACE %q has malformed flags", s)
	}
	mask, err := parseSDDLRights(f[2])
	if err != nil {
		return sddlACE{}, fmt.Errorf("ACE %q: %w", s, err)
	}
	ace.Mask = mask
	return ace, nil
}

// parseSDDLRights reads a mask written as a number ("0x1200a9") or as two-letter codes ("FRFW").
func parseSDDLRights(s string) (uint32, error) {
	if s != "" && s[0] >= '0' && s[0] <= '9' {
		v, err := strconv.ParseUint(s, 0, 32)
		if err != nil {
			return 0, fmt.Errorf("access mask %q: %w", s, err)
		}
		return uint32(v), nil
	}
	if len(s)%2 != 0 {
		return 0, fmt.Errorf("access mask %q is not a list of rights", s)
	}
	var mask uint32
	for i := 0; i < len(s); i += 2 {
		v, ok := sddlRights[s[i:i+2]]
		if !ok {
			return 0, fmt.Errorf("unknown access right %q", s[i:i+2])
		}
		mask |= v
	}
	return mask, nil
}
