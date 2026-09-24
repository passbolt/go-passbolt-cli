package util

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pterm/pterm"
)

type sampleItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestFprintJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := FprintJSON(&buf, sampleItem{ID: "1", Name: "ada", Age: 30}); err != nil {
		t.Fatalf("FprintJSON: %v", err)
	}
	want := "{\n  \"id\": \"1\",\n  \"name\": \"ada\",\n  \"age\": 30\n}\n"
	if got := buf.String(); got != want {
		t.Errorf("FprintJSON output mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestColumnFilteredMaps(t *testing.T) {
	items := []sampleItem{
		{ID: "1", Name: "ada", Age: 30},
		{ID: "2", Name: "grace", Age: 40},
	}

	t.Run("keeps only requested columns", func(t *testing.T) {
		got, err := columnFilteredMaps(items, []string{"id", "name"})
		if err != nil {
			t.Fatalf("columnFilteredMaps: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(got))
		}
		for i, m := range got {
			if _, ok := m["age"]; ok {
				t.Errorf("entry %d should not contain dropped column %q", i, "age")
			}
			if len(m) != 2 {
				t.Errorf("entry %d expected 2 keys, got %d: %v", i, len(m), m)
			}
		}
		if got[0]["id"] != "1" || got[0]["name"] != "ada" {
			t.Errorf("entry 0 unexpected values: %v", got[0])
		}
	})

	t.Run("unknown column is ignored", func(t *testing.T) {
		got, err := columnFilteredMaps(items, []string{"id", "nonexistent"})
		if err != nil {
			t.Fatalf("columnFilteredMaps: %v", err)
		}
		if len(got[0]) != 1 {
			t.Errorf("expected only the existing column to be kept, got %v", got[0])
		}
	})

	t.Run("empty items yields empty slice", func(t *testing.T) {
		got, err := columnFilteredMaps([]sampleItem{}, []string{"id"})
		if err != nil {
			t.Fatalf("columnFilteredMaps: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty result, got %v", got)
		}
	})
}

func TestPrintTable_UnknownColumn(t *testing.T) {
	items := []sampleItem{{ID: "1", Name: "ada"}}

	valueFn := func(it sampleItem, col string) (string, bool) {
		switch col {
		case "id":
			return it.ID, true
		case "name":
			return it.Name, true
		default:
			return "", false
		}
	}

	// Known columns render without error.
	if err := PrintTable([]string{"id", "name"}, items, valueFn); err != nil {
		t.Fatalf("PrintTable with known columns: %v", err)
	}

	// An unknown column returns the standard error.
	err := PrintTable([]string{"id", "bogus"}, items, valueFn)
	if err == nil {
		t.Fatal("expected error for unknown column, got nil")
	}
	if want := `unknown column: "bogus"`; err.Error() != want {
		t.Errorf("unexpected error: got %q, want %q", err.Error(), want)
	}
}

// encoding/json escapes C0 and U+2028 itself, but not C1, bidi overrides or
// zero-width runes - the same payloads Sanitize replaces on the table path.
func TestFprintJSONEscapesControlRunes(t *testing.T) {
	name := "x\u009b2K\x1b[2K\u202egpj.exe\u200b\U000E0001"

	var buf bytes.Buffer
	if err := FprintJSON(&buf, sampleItem{ID: "1", Name: name}); err != nil {
		t.Fatalf("FprintJSON: %v", err)
	}

	for _, r := range buf.String() {
		if !keepChar(r, utf8.RuneLen(r), true) {
			t.Errorf("control rune %U reached the stream: %q", r, buf.String())
		}
	}

	// Escaping, not stripping: the document still parses into the original value.
	var back sampleItem
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Name != name {
		t.Errorf("round-trip changed the value:\n got=%q\nwant=%q", back.Name, name)
	}
}

func TestEscapeJSONControlsLeavesPrintableUntouched(t *testing.T) {
	in := []byte("{\n  \"name\": \"Café Ünïcode 密码 \U0001F511\"\n}")
	if got := escapeJSONControls(in); got != string(in) {
		t.Errorf("got %q, want %q", got, string(in))
	}
}

func TestPrintJSONWritesToOut(t *testing.T) {
	var got bytes.Buffer
	orig := Out
	Out = NewSanitizingWriter(&got)
	t.Cleanup(func() { Out = orig })

	if err := PrintJSON(sampleItem{ID: "1", Name: "ada"}); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	if !strings.Contains(got.String(), `"name": "ada"`) {
		t.Errorf("PrintJSON did not reach Out: %q", got.String())
	}
}

// --json is lossless: the runes leave as \uXXXX, so nothing is replaced or warned about.
func TestPrintJSONReplacesNothing(t *testing.T) {
	var got bytes.Buffer
	orig := Out
	Out = NewSanitizingWriter(&got)
	t.Cleanup(func() { Out = orig; replacedRunes.Store(0) })

	replacedRunes.Store(0)
	if err := PrintJSON(sampleItem{ID: "1", Name: "x\u009b2K\x1b[2K\u202egpj.exe\u200b"}); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	if n := replacedRunes.Load(); n != 0 {
		t.Errorf("the JSON path replaced %d runes: %q", n, got.String())
	}
}

// A \n survives the stream writer by design, so a cell must be narrowed per value
// or a name forges a row. The header row is sanitized too: it is flag input.
func TestRenderTableSanitizesEveryCellAndWritesToOut(t *testing.T) {
	// The CLI disables styling at startup; do the same so the assertion sees only
	// the cell content and not pterm's own color escapes.
	pterm.DisableStyling()
	t.Cleanup(pterm.EnableStyling)

	var got bytes.Buffer
	orig := Out
	Out = NewSanitizingWriter(&got)
	t.Cleanup(func() { Out = orig })

	header := []string{"Name\x1b[2K", "ID"}
	data := pterm.TableData{
		header,
		{"real\nPermissions: owner", "1"},
		{"tab\there", "2"},
	}
	RenderTable(data)

	out := got.String()
	if rows := strings.Count(strings.TrimRight(out, "\n"), "\n") + 1; rows != 3 {
		t.Errorf("a cell forged a row, got %d rows: %q", rows, out)
	}
	if strings.ContainsAny(out, "\t\x1b") {
		t.Errorf("control byte reached the stream: %q", out)
	}
	if !strings.Contains(out, "real\uFFFDPermissions: owner") {
		t.Errorf("cell content lost: %q", out)
	}
	if header[0] != "Name\x1b[2K" {
		t.Errorf("RenderTable mutated the caller's slice: %q", header[0])
	}
}

func TestPrintTable_ValueFnError(t *testing.T) {
	// Guard the generic constraint compiles with a non-struct item type too.
	items := []string{"a", "b"}
	calls := 0
	valueFn := func(it string, col string) (string, bool) {
		calls++
		return it, true
	}
	if err := PrintTable([]string{"col"}, items, valueFn); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	if calls != len(items) {
		t.Errorf("expected valueFn called %d times, got %d", len(items), calls)
	}
}
