package util

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"cel.dev/cel-go/cel"
)

// testRow is a stand-in entity exercising the generic registry independent of
// any real Passbolt type.
type testRow struct {
	id     string
	name   string
	secret string
	active bool
}

func newTestRegistry() *ColumnRegistry[testRow] {
	return NewColumnRegistry("rows", []ColumnSpec[testRow]{
		{
			Name: "id", Aliases: []string{"ID"}, DefaultTable: true, CelType: cel.StringType,
			CelValue:   func(r testRow) any { return r.id },
			TableValue: func(r testRow) string { return r.id },
		},
		{
			Name: "name", Aliases: []string{"Name"}, DefaultTable: true, CelType: cel.StringType,
			CelValue:   func(r testRow) any { return r.name },
			TableValue: func(r testRow) string { return r.name },
		},
		{
			Name: "secret", Aliases: []string{"Secret"}, RequiresSecrets: true, CelType: cel.StringType,
			CelValue:   func(r testRow) any { return r.secret },
			TableValue: func(r testRow) string { return r.secret },
		},
		{
			Name: "active", Aliases: []string{"Active"}, CelType: cel.BoolType,
			CelValue:   func(r testRow) any { return r.active },
			TableValue: func(r testRow) string { return strconv.FormatBool(r.active) },
		},
	})
}

var testRows = []testRow{
	{id: "1", name: "alpha", secret: "s1", active: true},
	{id: "2", name: "beta", secret: "s2", active: false},
}

func TestColumnRegistry_Filter(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	cases := []struct {
		name    string
		expr    string
		wantIDs []string
	}{
		{"empty passes through", "", []string{"1", "2"}},
		{"canonical name", `name == "alpha"`, []string{"1"}},
		{"PascalCase alias", `Name == "alpha"`, []string{"1"}},
		{"bool field", `active`, []string{"1"}},
		{"secret field", `secret == "s2"`, []string{"2"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := r.Filter(ctx, testRows, c.expr)
			if err != nil {
				t.Fatalf("Filter(%q) error: %v", c.expr, err)
			}
			if len(got) != len(c.wantIDs) {
				t.Fatalf("Filter(%q) returned %d rows, want %d", c.expr, len(got), len(c.wantIDs))
			}
			for i, want := range c.wantIDs {
				if got[i].id != want {
					t.Errorf("Filter(%q)[%d].id = %q, want %q", c.expr, i, got[i].id, want)
				}
			}
		})
	}
}

func TestColumnRegistry_Filter_NoMatchError(t *testing.T) {
	r := newTestRegistry()
	_, err := r.Filter(context.Background(), testRows, `name == "nope"`)
	if err == nil {
		t.Fatal("expected error when no rows match")
	}
	if want := "no such rows found with filter"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q (entity noun)", err.Error(), want)
	}
}

func TestColumnRegistry_Filter_InvalidExpression(t *testing.T) {
	r := newTestRegistry()
	if _, err := r.Filter(context.Background(), testRows, `unknown_field == "x"`); err == nil {
		t.Fatal("expected compile error for unknown field")
	}
}

func TestColumnRegistry_CelEnvOptions_CountsCanonicalAndAliases(t *testing.T) {
	r := newTestRegistry()
	// 4 columns, each with exactly 1 alias → 8 variables registered.
	if got, want := len(r.CelEnvOptions()), 8; got != want {
		t.Errorf("CelEnvOptions length = %d, want %d (canonical + aliases)", got, want)
	}
}

func TestColumnRegistry_RequiresSecrets(t *testing.T) {
	r := newTestRegistry()
	cases := []struct {
		cols []string
		want bool
	}{
		{[]string{"id", "name"}, false},
		{[]string{"id", "secret"}, true},
		{[]string{"active"}, false},
	}
	for _, c := range cases {
		if got := r.RequiresSecrets(c.cols); got != c.want {
			t.Errorf("RequiresSecrets(%v) = %v, want %v", c.cols, got, c.want)
		}
	}
}

func TestColumnRegistry_SecretCelNames(t *testing.T) {
	r := newTestRegistry()
	have := map[string]bool{}
	for _, n := range r.SecretCelNames() {
		have[n] = true
	}
	for _, want := range []string{"secret", "Secret"} {
		if !have[want] {
			t.Errorf("SecretCelNames missing %q", want)
		}
	}
	if have["name"] {
		t.Error("SecretCelNames must not contain non-secret column 'name'")
	}
}

func TestColumnRegistry_DefaultTableColumns(t *testing.T) {
	r := newTestRegistry()
	got := r.DefaultTableColumns()
	want := []string{"id", "name"}
	if len(got) != len(want) {
		t.Fatalf("DefaultTableColumns = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("DefaultTableColumns[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	// Defensive copy: mutating the result must not affect later calls.
	got[0] = "mutated"
	if again := r.DefaultTableColumns(); again[0] != "id" {
		t.Errorf("DefaultTableColumns not defensively copied: again[0] = %q", again[0])
	}
}

func TestColumnRegistry_TableValue(t *testing.T) {
	r := newTestRegistry()
	row := testRows[1]
	if v, ok := r.TableValue(row, "name"); !ok || v != "beta" {
		t.Errorf("TableValue(name) = (%q, %v), want (\"beta\", true)", v, ok)
	}
	if v, ok := r.TableValue(row, "active"); !ok || v != "false" {
		t.Errorf("TableValue(active) = (%q, %v), want (\"false\", true)", v, ok)
	}
	if _, ok := r.TableValue(row, "missing"); ok {
		t.Error("TableValue(missing) should report ok=false for unknown column")
	}
}
