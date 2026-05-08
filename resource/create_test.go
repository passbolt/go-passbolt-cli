package resource

import (
	"reflect"
	"testing"
)

func TestParseKeyValue_Strings(t *testing.T) {
	cases := []struct {
		in    string
		key   string
		value any
	}{
		{"name=hello", "name", "hello"},
		{"username=alice@example.com", "username", "alice@example.com"},
		{"description=multi word value", "description", "multi word value"},
		{"empty=", "empty", ""},
		{"key=value=with=equals", "key", "value=with=equals"},
		{"name= leading space", "name", " leading space"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			k, v, err := parseKeyValue(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if k != tc.key || v != tc.value {
				t.Errorf("parseKeyValue(%q) = (%q, %v), want (%q, %v)", tc.in, k, v, tc.key, tc.value)
			}
		})
	}
}

func TestParseKeyValue_JSONArray(t *testing.T) {
	k, v, err := parseKeyValue(`custom_fields=[{"id":"abc","type":"text"}]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "custom_fields" {
		t.Errorf("key = %q, want custom_fields", k)
	}
	want := []any{map[string]any{"id": "abc", "type": "text"}}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("value = %#v, want %#v", v, want)
	}
}

func TestParseKeyValue_JSONObject(t *testing.T) {
	k, v, err := parseKeyValue(`totp={"secret_key":"JBSWY3DPEHPK3PXP","digits":6}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "totp" {
		t.Errorf("key = %q, want totp", k)
	}
	want := map[string]any{"secret_key": "JBSWY3DPEHPK3PXP", "digits": float64(6)}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("value = %#v, want %#v", v, want)
	}
}

func TestParseKeyValue_JSONLikeButInvalidStaysAsString(t *testing.T) {
	// Starts with '[' but isn't valid JSON; must fall back to literal string,
	// not error — the server schema layer surfaces validation errors later.
	k, v, err := parseKeyValue(`weird=[not actually json`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "weird" {
		t.Errorf("key = %q", k)
	}
	if s, ok := v.(string); !ok || s != "[not actually json" {
		t.Errorf("value should be the literal string, got %T %v", v, v)
	}
}

func TestParseKeyValue_LeadingWhitespaceTriggersJSONDetect(t *testing.T) {
	// The detection trims whitespace before checking the prefix, so values
	// with leading whitespace before the JSON open-bracket still parse as JSON.
	_, v, err := parseKeyValue("k=  [1,2,3]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(v, []any{float64(1), float64(2), float64(3)}) {
		t.Errorf("value = %#v, want [1 2 3]", v)
	}
}

func TestParseKeyValue_NoEquals(t *testing.T) {
	if _, _, err := parseKeyValue("noequals"); err == nil {
		t.Error("expected error for input without '='")
	}
}
