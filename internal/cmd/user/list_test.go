package user

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestUserListCmd builds a Cobra command shaped like UserListCmd but with
// the persistent flags from listCmd (--json, --filter) declared locally so
// parseUserListFlags can read them without depending on the real command
// tree.
func newTestUserListCmd() *cobra.Command {
	c := &cobra.Command{Use: "user"}
	f := c.Flags()
	f.Bool("include-mfa", false, "")
	f.Bool("mfa-enabled", false, "")
	f.Bool("scim-entry", false, "")
	f.StringArrayP("group", "g", []string{}, "")
	f.StringArrayP("resource", "r", []string{}, "")
	f.StringP("search", "s", "", "")
	f.BoolP("admin", "a", false, "")
	f.StringArrayP("column", "c", userColumns.DefaultTableColumns(), "")
	f.BoolP("json", "j", false, "")
	f.String("filter", "", "")
	return c
}

func TestParseUserListFlags_AcceptsAliasedColumn(t *testing.T) {
	// Sanity check that the normalization path runs:
	// PascalCase aliases should land as their canonical snake_case form.
	c := newTestUserListCmd()
	if err := c.ParseFlags([]string{"--column", "FirstName", "--column", "ID"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	cfg, err := parseUserListFlags(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"first_name", "id"}
	if len(cfg.columns) != len(want) {
		t.Fatalf("columns len: got %d, want %d (%v)", len(cfg.columns), len(want), cfg.columns)
	}
	for i := range want {
		if cfg.columns[i] != want[i] {
			t.Errorf("columns[%d] = %q, want %q", i, cfg.columns[i], want[i])
		}
	}
}

func TestParseUserListFlags_RejectsUnknownColumn(t *testing.T) {
	c := newTestUserListCmd()
	if err := c.ParseFlags([]string{"--column", "nope"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	_, err := parseUserListFlags(c)
	if err == nil {
		t.Fatal("expected error from unknown column")
	}
	if !strings.Contains(err.Error(), "unknown column") {
		t.Errorf("error %q should mention 'unknown column'", err.Error())
	}
}
