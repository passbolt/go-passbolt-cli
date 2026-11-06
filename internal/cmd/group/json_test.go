package group

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
)

// GroupJSONOutput's Deleted/UserCount must emit even when false/0 so that
// --filter 'deleted == false' / 'user_count == 0' isn't blocked by omitempty.
func TestGroupJSONOutput_ScalarFieldsAlwaysEmitted(t *testing.T) {
	out := GroupJSONOutput{ID: new("g1"), Name: new("eng"), Deleted: false, UserCount: 0}
	got, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"deleted":false`, `"user_count":0`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %q in output %s", want, got)
		}
	}
}

func TestPrintJSONGroups_PopulatesFields(t *testing.T) {
	groups := []api.Group{
		{
			ID:        "g1",
			Name:      "engineering",
			Created:   &api.Time{},
			Modified:  &api.Time{},
			Deleted:   false,
			UserCount: 5,
		},
	}

	out := captureOutput(t, func() {
		if err := printJSONGroups(groups, false, nil); err != nil {
			t.Fatalf("printJSONGroups: %v", err)
		}
	})

	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\nraw:\n%s", err, out)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 group, got %d", len(got))
	}
	if got[0]["id"] != "g1" || got[0]["name"] != "engineering" {
		t.Errorf("group[0] = %v", got[0])
	}
	if got[0]["user_count"] != float64(5) {
		t.Errorf("user_count = %v, want 5", got[0]["user_count"])
	}
}

func TestPrintJSONGroups_ColumnFilterRestrictsKeys(t *testing.T) {
	groups := []api.Group{
		{ID: "g1", Name: "engineering", Created: &api.Time{}, Modified: &api.Time{}, UserCount: 5},
	}

	out := captureOutput(t, func() {
		if err := printJSONGroups(groups, true, []string{"id", "name"}); err != nil {
			t.Fatalf("printJSONGroups: %v", err)
		}
	})

	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\nraw:\n%s", err, out)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 group, got %d", len(got))
	}
	if _, ok := got[0]["user_count"]; ok {
		t.Errorf("user_count should be filtered out, got %v", got[0])
	}
	if got[0]["name"] != "engineering" {
		t.Errorf("name = %v, want engineering", got[0]["name"])
	}
}

// Pins the JSON struct contract for the get path (GroupGet builds this same
// shape inline). NOTE: tests the struct's omitempty/tag behavior, not the
// command — name + nested users array present, deleted/user_count
// (non-omitempty) still appear, id/timestamps omitted. End-to-end output is
// covered by the integration get-roundtrip scripts.
func TestGroupGetJSONContract(t *testing.T) {
	out := GroupJSONOutput{
		Name: new("eng"),
		Users: []GroupUserMembershipJSONOutput{
			{ID: new("u1"), Username: new("ada@x"), FirstName: new("Ada"), LastName: new("L"), IsGroupManager: new(true)},
		},
	}
	m := marshalToMap(t, out)
	assertKeySet(t, m, map[string]bool{"name": true, "users": true, "deleted": true, "user_count": true})
	for _, k := range []string{"id", "created_timestamp", "modified_timestamp"} {
		if _, ok := m[k]; ok {
			t.Errorf("key %q should be omitted in get JSON", k)
		}
	}
	users, ok := m["users"].([]any)
	if !ok || len(users) != 1 {
		t.Fatalf("users = %T (%v), want array of 1", m["users"], m["users"])
	}
	member, ok := users[0].(map[string]any)
	if !ok {
		t.Fatalf("users[0] = %T, want object", users[0])
	}
	if _, ok := member["is_group_manager"].(bool); !ok {
		t.Errorf("is_group_manager = %T, want bool", member["is_group_manager"])
	}
	if member["username"] != "ada@x" {
		t.Errorf("username = %v, want ada@x", member["username"])
	}
}

func TestPrintJSONGroups_EscapesSpecialChars(t *testing.T) {
	weird := "a\"b\\c\nd<eé"
	groups := []api.Group{{ID: "g1", Name: weird, Created: &api.Time{}, Modified: &api.Time{}}}
	out := captureOutput(t, func() {
		if err := printJSONGroups(groups, false, nil); err != nil {
			t.Fatalf("printJSONGroups: %v", err)
		}
	})
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw:\n%s", err, out)
	}
	if got[0]["name"] != weird {
		t.Errorf("name round-trip = %q, want %q", got[0]["name"], weird)
	}
}

func TestPrintJSONGroups_EmptyListIsArray(t *testing.T) {
	out := captureOutput(t, func() {
		if err := printJSONGroups(nil, false, nil); err != nil {
			t.Fatalf("printJSONGroups: %v", err)
		}
	})
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("empty list output = %q, want []", strings.TrimSpace(out))
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if got == nil {
		t.Error("empty list should decode to an empty array, not null")
	}
}

func marshalToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func assertKeySet(t *testing.T, m map[string]any, want map[string]bool) {
	t.Helper()
	for k := range m {
		if !want[k] {
			t.Errorf("unexpected key %q in JSON output", k)
		}
	}
	for k := range want {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %q in JSON output", k)
		}
	}
}

// captureOutput swaps util.Out for a buffer and returns whatever fn wrote to it.
// It captures util.Out rather than os.Stdout because that is the stream the CLI
// prints on: util.Out wraps os.Stdout once, at init, so redirecting the file
// afterwards would miss the writes.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	orig := util.Out
	util.Out = util.NewSanitizingWriter(&buf)
	t.Cleanup(func() { util.Out = orig })

	fn()
	return buf.String()
}
