package folder

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
)

// FolderJSONOutput.Personal must emit even when false so that
// --filter 'personal == false' isn't blocked by omitempty stripping.
func TestFolderJSONOutput_PersonalAlwaysEmitted(t *testing.T) {
	out := FolderJSONOutput{ID: new("f1"), Name: new("shared"), Personal: false}
	got, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(got), `"personal":false`) {
		t.Errorf("missing \"personal\":false in output %s", got)
	}
}

func TestPrintJSONFolders_PopulatesFields(t *testing.T) {
	folders := []api.Folder{
		{
			ID:             "f1",
			FolderParentID: "p1",
			Name:           "shared",
			Created:        &api.Time{},
			Modified:       &api.Time{},
			Personal:       false,
		},
		{
			ID:       "f2",
			Name:     "private",
			Created:  &api.Time{},
			Modified: &api.Time{},
			Personal: true,
		},
	}

	out := captureOutput(t, func() {
		if err := printJSONFolders(folders, false, nil); err != nil {
			t.Fatalf("printJSONFolders: %v", err)
		}
	})

	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\nraw:\n%s", err, out)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(got))
	}
	if got[0]["id"] != "f1" || got[0]["name"] != "shared" {
		t.Errorf("folder[0] = %v", got[0])
	}
	if got[0]["personal"] != false {
		t.Errorf("folder[0].personal = %v, want false", got[0]["personal"])
	}
	if got[1]["personal"] != true {
		t.Errorf("folder[1].personal = %v, want true", got[1]["personal"])
	}
}

func TestPrintJSONFolders_ColumnFilterRestrictsKeys(t *testing.T) {
	folders := []api.Folder{
		{ID: "f1", Name: "shared", FolderParentID: "p1", Created: &api.Time{}, Modified: &api.Time{}},
	}

	out := captureOutput(t, func() {
		if err := printJSONFolders(folders, true, []string{"id", "name"}); err != nil {
			t.Fatalf("printJSONFolders: %v", err)
		}
	})

	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\nraw:\n%s", err, out)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(got))
	}
	if _, ok := got[0]["folder_parent_id"]; ok {
		t.Errorf("folder_parent_id should be filtered out, got %v", got[0])
	}
	if got[0]["name"] != "shared" {
		t.Errorf("name = %v, want shared", got[0]["name"])
	}
}

// Pins the JSON struct contract for the get path (FolderGet builds this same
// subset inline). NOTE: tests the struct's omitempty/tag behavior, not the
// command — folder_parent_id + name present, personal (non-omitempty) still
// appears, id/timestamps omitted. End-to-end output is covered by the
// integration get-roundtrip scripts.
func TestFolderGetJSONContract(t *testing.T) {
	out := FolderJSONOutput{FolderParentID: new("p1"), Name: new("shared")}
	m := marshalToMap(t, out)
	assertKeySet(t, m, map[string]bool{"folder_parent_id": true, "name": true, "personal": true})
	for _, k := range []string{"id", "created_timestamp", "modified_timestamp"} {
		if _, ok := m[k]; ok {
			t.Errorf("key %q should be omitted in get JSON", k)
		}
	}
}

func TestPrintJSONFolders_EscapesSpecialChars(t *testing.T) {
	weird := "a\"b\\c\nd<eé"
	folders := []api.Folder{{ID: "f1", Name: weird, Created: &api.Time{}, Modified: &api.Time{}}}
	out := captureOutput(t, func() {
		if err := printJSONFolders(folders, false, nil); err != nil {
			t.Fatalf("printJSONFolders: %v", err)
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

func TestPrintJSONFolders_EmptyListIsArray(t *testing.T) {
	out := captureOutput(t, func() {
		if err := printJSONFolders(nil, false, nil); err != nil {
			t.Fatalf("printJSONFolders: %v", err)
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
