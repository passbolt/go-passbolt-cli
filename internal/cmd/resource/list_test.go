package resource

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestResourceListCmd builds a Cobra command shaped like ResourceListCmd but
// with the persistent flags from listCmd (--json, --filter) declared locally so
// parseResourceListFlags can read them without depending on the real command
// tree.
func newTestResourceListCmd() *cobra.Command {
	c := &cobra.Command{Use: "resource"}
	f := c.Flags()
	f.Bool("favorite", false, "")
	f.Bool("own", false, "")
	f.StringP("group", "g", "", "")
	f.StringArrayP("folder", "f", []string{}, "")
	f.StringArrayP("column", "c", resourceColumns.DefaultTableColumns(), "")
	f.BoolP("json", "j", false, "")
	f.String("filter", "", "")
	return c
}

func TestParseResourceListFlags_NormalizesAliasedColumns(t *testing.T) {
	c := newTestResourceListCmd()
	if err := c.ParseFlags([]string{"--column", "Name", "--column", "Username", "--column", "ID"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	cfg, err := parseResourceListFlags(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"name", "username", "id"}
	if len(cfg.columns) != len(want) {
		t.Fatalf("columns: got %v, want %v", cfg.columns, want)
	}
	for i := range want {
		if cfg.columns[i] != want[i] {
			t.Errorf("columns[%d] = %q, want %q", i, cfg.columns[i], want[i])
		}
	}
	if !cfg.columnsChanged {
		t.Error("columnsChanged should be true when --column is set")
	}
}

func TestParseResourceListFlags_RejectsUnknownColumn(t *testing.T) {
	c := newTestResourceListCmd()
	if err := c.ParseFlags([]string{"--column", "nope"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	_, err := parseResourceListFlags(c)
	if err == nil {
		t.Fatal("expected error from unknown column")
	}
	if !strings.Contains(err.Error(), "unknown column") {
		t.Errorf("error %q should mention 'unknown column'", err.Error())
	}
}

func TestParseResourceListFlags_MapsFilterFlags(t *testing.T) {
	c := newTestResourceListCmd()
	args := []string{
		"--favorite",
		"--own",
		"--group", "group-id",
		"--folder", "f1",
		"--folder", "f2",
		"--json",
		"--filter", "name == 'x'",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	cfg, err := parseResourceListFlags(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.favorite {
		t.Error("favorite should be true")
	}
	if !cfg.own {
		t.Error("own should be true")
	}
	if cfg.group != "group-id" {
		t.Errorf("group = %q, want group-id", cfg.group)
	}
	if len(cfg.folderParents) != 2 || cfg.folderParents[0] != "f1" || cfg.folderParents[1] != "f2" {
		t.Errorf("folderParents = %v, want [f1 f2]", cfg.folderParents)
	}
	if !cfg.jsonOutput {
		t.Error("jsonOutput should be true")
	}
	if cfg.celFilter != "name == 'x'" {
		t.Errorf("celFilter = %q, want \"name == 'x'\"", cfg.celFilter)
	}
}

func TestParseResourceListFlags_DefaultsColumnsNotChanged(t *testing.T) {
	c := newTestResourceListCmd()
	if err := c.ParseFlags(nil); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	cfg, err := parseResourceListFlags(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.columnsChanged {
		t.Error("columnsChanged should be false when --column is not set")
	}
	if len(cfg.columns) == 0 {
		t.Error("columns should default to the registry default set")
	}
}

// TestFormatSkippedTypes pins the notice shown for resources this build cannot decrypt. The
// ordering assertion matters: the tally is a map, so the lines used to come out in random order.
func TestFormatSkippedTypes(t *testing.T) {
	tests := []struct {
		name     string
		counts   map[string]int
		want     string
		contains []string
	}{
		{
			name:   "nothing skipped renders nothing",
			counts: nil,
			want:   "",
		},
		{
			name:   "empty map renders nothing",
			counts: map[string]int{},
			want:   "",
		},
		{
			name:   "one resource uses the singular",
			counts: map[string]int{"v5-quantum": 1},
			want: "Warning: 1 resource ignored: their Resource type is unknown to this version of " +
				"go-passbolt-cli or disabled on the server\n" +
				"  - v5-quantum: 1\n" +
				"Update go-passbolt-cli to include the types it does not know yet\n",
		},
		{
			name:   "several slugs are totalled and sorted",
			counts: map[string]int{"v5-quantum": 2, "v5-brandnew": 1},
			want: "Warning: 3 resources ignored: their Resource type is unknown to this version of " +
				"go-passbolt-cli or disabled on the server\n" +
				"  - v5-brandnew: 1\n" +
				"  - v5-quantum: 2\n" +
				"Update go-passbolt-cli to include the types it does not know yet\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatSkippedTypes(tc.counts); got != tc.want {
				t.Errorf("formatSkippedTypes() =\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}
