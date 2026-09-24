package resource

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
)

// ResourceJSONOutput's Deleted/Expired fields must emit even when false so that
// --filter 'deleted == false' isn't blocked by omitempty-style stripping.
// Regression guard mirroring the user lifecycle-field test.
func TestResourceJSONOutput_BoolFieldsAlwaysEmitted(t *testing.T) {
	out := ResourceJSONOutput{
		ID:      new("r1"),
		Name:    new("server"),
		Deleted: false,
		Expired: false,
	}
	got, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"deleted":false`, `"expired":false`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %q in output %s", want, got)
		}
	}
}

// printJSONResources must map decryptedResource onto ResourceJSONOutput,
// preserving the decrypted fields and resource metadata.
func TestPrintJSONResources_PopulatesFields(t *testing.T) {
	decrypted := []decryptedResource{
		{
			resource: api.Resource{
				ID:             "r1",
				FolderParentID: "f1",
				ResourceTypeID: "type-1",
				Created:        &api.Time{},
				Modified:       &api.Time{},
				Deleted:        false,
			},
			name:        "server",
			username:    "admin",
			uri:         "https://example.com",
			password:    "secret",
			description: "prod box",
		},
	}

	out := captureOutput(t, func() {
		if err := printJSONResources(decrypted, false, nil); err != nil {
			t.Fatalf("printJSONResources: %v", err)
		}
	})

	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\nraw:\n%s", err, out)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(got))
	}
	rec := got[0]
	for field, want := range map[string]string{
		"id":       "r1",
		"name":     "server",
		"username": "admin",
		"uri":      "https://example.com",
		"password": "secret",
	} {
		if rec[field] != want {
			t.Errorf("%s = %v, want %v", field, rec[field], want)
		}
	}
}

// When the user selects a subset of columns, only those keys should survive in
// the JSON output (via util.PrintJSONColumnFiltered).
func TestPrintJSONResources_ColumnFilterRestrictsKeys(t *testing.T) {
	decrypted := []decryptedResource{
		{
			resource: api.Resource{
				ID:       "r1",
				Created:  &api.Time{},
				Modified: &api.Time{},
			},
			name:     "server",
			password: "secret",
		},
	}

	out := captureOutput(t, func() {
		if err := printJSONResources(decrypted, true, []string{"id", "name"}); err != nil {
			t.Fatalf("printJSONResources: %v", err)
		}
	})

	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\nraw:\n%s", err, out)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(got))
	}
	if _, ok := got[0]["password"]; ok {
		t.Errorf("password should be filtered out, got %v", got[0])
	}
	if got[0]["name"] != "server" {
		t.Errorf("name = %v, want server", got[0]["name"])
	}
}

// Pins the JSON struct contract for the get path (ResourceGet builds this same
// subset inline). NOTE: this tests the struct's omitempty/tag behavior, not the
// command itself — it asserts id/timestamps are omitted while deleted/expired
// (non-omitempty) still appear as false. End-to-end command output is covered by
// the integration get-roundtrip scripts.
func TestResourceGetJSONContract(t *testing.T) {
	out := ResourceJSONOutput{
		FolderParentID: new("f1"),
		Name:           new("server"),
		Username:       new("admin"),
		URI:            new("https://example.com"),
		Password:       new("secret"),
		Description:    new("d"),
	}
	m := marshalToMap(t, out)

	want := map[string]bool{
		"folder_parent_id": true, "name": true, "username": true,
		"uri": true, "password": true, "description": true,
		"deleted": true, "expired": true,
	}
	assertKeySet(t, m, want)
	for _, k := range []string{"id", "created_timestamp", "modified_timestamp", "resource_type_id", "metadata", "secret", "expired_at"} {
		if _, ok := m[k]; ok {
			t.Errorf("key %q should be omitted in get JSON", k)
		}
	}
}

// Special characters in a field must be JSON-escaped so the output stays valid
// and round-trips to the exact original value.
func TestPrintJSONResources_EscapesSpecialChars(t *testing.T) {
	weird := "a\"b\\c\nd<eé"
	decrypted := []decryptedResource{
		{
			resource: api.Resource{ID: "r1", Created: &api.Time{}, Modified: &api.Time{}},
			name:     weird,
		},
	}
	out := captureOutput(t, func() {
		if err := printJSONResources(decrypted, false, nil); err != nil {
			t.Fatalf("printJSONResources: %v", err)
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

// An empty result set must serialize to [] (not null) so array consumers (jq)
// don't break.
func TestPrintJSONResources_EmptyListIsArray(t *testing.T) {
	out := captureOutput(t, func() {
		if err := printJSONResources(nil, false, nil); err != nil {
			t.Fatalf("printJSONResources: %v", err)
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

// marshalToMap marshals v and decodes it into a generic map so tests can assert
// key presence and JSON value types.
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

// assertKeySet fails if m has any key outside want, or is missing any key in want.
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
