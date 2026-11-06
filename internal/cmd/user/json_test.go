package user

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
)

// UserJSONOutput's lifecycle fields (active, deleted, disabled) must emit
// even when false so that --filter 'active == false' isn't blocked by
// omitempty-style stripping. Regression guard.
func TestUserJSONOutput_LifecycleFieldsAlwaysEmitted(t *testing.T) {
	out := UserJSONOutput{
		ID:       new("u1"),
		Username: new("ada@passbolt.com"),
		Active:   false,
		Deleted:  false,
		Disabled: false,
	}
	got, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"active":false`, `"deleted":false`, `"disabled":false`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %q in output %s", want, got)
		}
	}
}

// printJSONUsers must map api.User onto UserJSONOutput with the lifecycle
// flags correctly derived — Active/Deleted as direct passthrough, Disabled
// from User.Disabled != nil. Catches struct-literal-side bugs that the JSON
// tag test above can't.
func TestPrintJSONUsers_PopulatesLifecycleFields(t *testing.T) {
	disabledTime := api.Time{}
	users := []api.User{
		{
			ID: "active-user", Username: "ada@passbolt.com",
			Profile: &api.Profile{FirstName: "Ada", LastName: "Lovelace"},
			Role:    &api.Role{Name: "user"},
			Created: &api.Time{}, Modified: &api.Time{},
			Active: true, Deleted: false, Disabled: nil,
		},
		{
			ID: "disabled-user", Username: "grace@passbolt.com",
			Profile: &api.Profile{FirstName: "Grace", LastName: "Hopper"},
			Role:    &api.Role{Name: "user"},
			Created: &api.Time{}, Modified: &api.Time{},
			Active: true, Deleted: false, Disabled: &disabledTime,
		},
		{
			ID: "deleted-user", Username: "old@passbolt.com",
			Profile: &api.Profile{FirstName: "Old", LastName: "Account"},
			Role:    &api.Role{Name: "user"},
			Created: &api.Time{}, Modified: &api.Time{},
			Active: false, Deleted: true, Disabled: nil,
		},
	}

	out := captureOutput(t, func() {
		if err := printJSONUsers(users, false, nil); err != nil {
			t.Fatalf("printJSONUsers: %v", err)
		}
	})

	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\nraw:\n%s", err, out)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 users in output, got %d", len(got))
	}

	cases := []struct {
		id              string
		active, deleted bool
		disabled        bool
	}{
		{"active-user", true, false, false},
		{"disabled-user", true, false, true},
		{"deleted-user", false, true, false},
	}
	for _, c := range cases {
		var record map[string]any
		for _, r := range got {
			if r["id"] == c.id {
				record = r
				break
			}
		}
		if record == nil {
			t.Errorf("no record for id=%q", c.id)
			continue
		}
		if record["active"] != c.active {
			t.Errorf("%s: active = %v, want %v", c.id, record["active"], c.active)
		}
		if record["deleted"] != c.deleted {
			t.Errorf("%s: deleted = %v, want %v", c.id, record["deleted"], c.deleted)
		}
		if record["disabled"] != c.disabled {
			t.Errorf("%s: disabled = %v, want %v", c.id, record["disabled"], c.disabled)
		}
	}
}

// Pins the JSON struct contract for the get path (UserGet builds this same
// subset inline). NOTE: tests the struct's omitempty/tag behavior, not the
// command — username/first_name/last_name/role present, lifecycle flags
// (non-omitempty) still appear as false, id/timestamps omitted. End-to-end
// output is covered by the integration get-roundtrip scripts.
func TestUserGetJSONContract(t *testing.T) {
	out := UserJSONOutput{
		Username:  new("ada@passbolt.com"),
		FirstName: new("Ada"),
		LastName:  new("Lovelace"),
		Role:      new("user"),
	}
	m := marshalToMap(t, out)
	assertKeySet(t, m, map[string]bool{
		"username": true, "first_name": true, "last_name": true, "role": true,
		"active": true, "deleted": true, "disabled": true,
	})
	for _, k := range []string{"id", "created_timestamp", "modified_timestamp"} {
		if _, ok := m[k]; ok {
			t.Errorf("key %q should be omitted in get JSON", k)
		}
	}
}

func TestPrintJSONUsers_EscapesSpecialChars(t *testing.T) {
	weird := "a\"b\\c\nd<eé"
	users := []api.User{
		{
			ID:       "u1",
			Username: "ada@passbolt.com",
			Profile:  &api.Profile{FirstName: weird, LastName: "Lovelace"},
			Role:     &api.Role{Name: "user"},
			Created:  &api.Time{},
			Modified: &api.Time{},
		},
	}
	out := captureOutput(t, func() {
		if err := printJSONUsers(users, false, nil); err != nil {
			t.Fatalf("printJSONUsers: %v", err)
		}
	})
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw:\n%s", err, out)
	}
	if got[0]["first_name"] != weird {
		t.Errorf("first_name round-trip = %q, want %q", got[0]["first_name"], weird)
	}
}

func TestPrintJSONUsers_EmptyListIsArray(t *testing.T) {
	out := captureOutput(t, func() {
		if err := printJSONUsers(nil, false, nil); err != nil {
			t.Fatalf("printJSONUsers: %v", err)
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
