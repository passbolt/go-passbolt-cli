package util

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseSDDLDACL(t *testing.T) {
	tests := []struct {
		name string
		sddl string
		want []sddlACE
	}{
		{
			name: "privateSD for a directory",
			sddl: "D:P(A;OICI;FA;;;S-1-5-21-1-2-3-1001)(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)",
			want: []sddlACE{
				{Type: "A", Flags: "OICI", Mask: 0x1F01FF, SID: "S-1-5-21-1-2-3-1001"},
				{Type: "A", Flags: "OICI", Mask: 0x1F01FF, SID: "SY"},
				{Type: "A", Flags: "OICI", Mask: 0x1F01FF, SID: "BA"},
			},
		},
		{
			name: "hex mask, inherited",
			sddl: "D:AI(A;ID;0x1200a9;;;BU)(D;;FW;;;WD)",
			want: []sddlACE{
				{Type: "A", Flags: "ID", Mask: 0x1200A9, SID: "BU"},
				{Type: "D", Mask: 0x120116, SID: "WD"},
			},
		},
		{
			name: "decomposed rights",
			sddl: "D:(A;;CCDCRC;;;WD)",
			want: []sddlACE{{Type: "A", Mask: 0x1 | 0x2 | 0x20000, SID: "WD"}},
		},
		{name: "empty DACL", sddl: "D:P", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSDDLDACL(tt.sddl)
			if err != nil {
				t.Fatalf("parseSDDLDACL(%q) = %v", tt.sddl, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSDDLDACL(%q)\n got %+v\nwant %+v", tt.sddl, got, tt.want)
			}
		})
	}
}

func TestParseSDDLDACL_InheritOnly(t *testing.T) {
	aces, err := parseSDDLDACL("D:(A;CIOI;FA;;;WD)(A;OICIIO;FA;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	// "CIOI" contains "IO" across a code boundary; only the second ACE is inherit-only.
	if aces[0].inheritOnly() || !aces[1].inheritOnly() {
		t.Errorf("inheritOnly = %v, %v; want false, true", aces[0].inheritOnly(), aces[1].inheritOnly())
	}
}

// TestParseSDDLDACL_FailsClosed: looseTrustee treats every error here as "exposed".
func TestParseSDDLDACL_FailsClosed(t *testing.T) {
	for _, sddl := range []string{
		"",                       // what SECURITY_DESCRIPTOR.String returns on failure
		"O:BAG:SY",               // no DACL
		"O:BAG:SYD:(A;;FA;;;SY)", // other sections
		"D:(A;;FA;;;SY)S:(AU;SA;FA;;;WD)",
		"D:X(A;;FA;;;SY)", // unknown DACL flag
		`D:(XA;;FR;;;WD;(@User.Title == "a)b("))(A;;FA;;;SY)`, // conditional
		"D:(XD;;FR;;;WD;(Member_of {SID(BA)}))",
		"D:NO_ACCESS_CONTROL",     // NULL DACL
		"D:(A;;FA;;;WD",           // unterminated
		"D:(A;;FA;;;WD))",         // unbalanced
		"D:(A;;FA;;;WD)junk",      // text between ACEs
		"D:(A;;FA;;)",             // too few fields
		"D:(A;;FA;;;)",            // no SID
		"D:(A;;ZZ;;;WD)",          // unknown right
		"D:(A;;FAF;;;WD)",         // odd-length rights
		"D:(A;;0x1ffffffff;;;WD)", // mask overflows 32 bits
		"D:(A;CIO;FA;;;WD)",       // odd-length flags
		"D:(;;FA;;;WD)",           // no type
	} {
		if aces, err := parseSDDLDACL(sddl); err == nil {
			t.Errorf("parseSDDLDACL(%q) = %+v, want an error", sddl, aces)
		}
	}
	if _, err := parseSDDLDACL("D:NO_ACCESS_CONTROL"); !errors.Is(err, errNoDACL) {
		t.Errorf("NULL DACL: err = %v, want errNoDACL", err)
	}
}
