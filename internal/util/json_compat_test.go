package util

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Go 1.27 reimplemented encoding/json on top of encoding/json/v2, keeping the v1
// API and v1 semantics. `omitempty` is one of the places the two differ: v2 omits
// empty JSON values and offers `omitzero` separately, while v1 omits empty Go
// values. PermissionJSONOutput relies on the v1 reading for every field, and its
// output is user-visible in `passbolt get resource --json`, so the exact bytes are
// pinned here.
//
// Deliberately not behind a build tag: this must hold under both backends. Run it
// both ways:
//
//	go test ./internal/util/
//	GOEXPERIMENT=nojsonv2 go test ./internal/util/

// TestPermissionJSONOutput_OmitEmptyOnNilPointers pins that unset (nil) pointer
// fields disappear from the output entirely rather than serializing as null.
func TestPermissionJSONOutput_OmitEmptyOnNilPointers(t *testing.T) {
	t.Parallel()

	id := "d4c0e643-3967-443b-93b3-102d902c4510"
	// Only ID is set; every other field is left nil.
	got, err := json.Marshal(PermissionJSONOutput{ID: &id})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	const want = `{"id":"d4c0e643-3967-443b-93b3-102d902c4510"}`
	if string(got) != want {
		t.Errorf("Marshal = %s, want %s", got, want)
	}
	if strings.Contains(string(got), "null") {
		t.Errorf("nil pointer fields serialized as null instead of being omitted: %s", got)
	}
}

// TestPermissionJSONOutput_ZeroValuesBehindPointersAreKept pins the other half of
// the contract, and is the case most at risk from an omitempty semantic change: a
// pointer to a zero value is NOT empty, so the field must still be emitted.
//
// Type is the one that matters. Passbolt permission type 0 is meaningful, and if
// it were ever dropped from the output a consumer could not tell "no permission"
// from "field absent".
func TestPermissionJSONOutput_ZeroValuesBehindPointersAreKept(t *testing.T) {
	t.Parallel()

	var (
		empty    string
		zeroType int
		zeroTime time.Time
	)
	got, err := json.Marshal(PermissionJSONOutput{
		Aco:              &empty,
		Type:             &zeroType,
		CreatedTimestamp: &zeroTime,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	const want = `{"aco":"","type":0,"created_timestamp":"0001-01-01T00:00:00Z"}`
	if string(got) != want {
		t.Errorf("Marshal = %s, want %s\n(a pointer to a zero value is not empty and must still be emitted)", got, want)
	}
}

// TestPermissionsToJSONOutput_EmptyInputIsEmptyArray pins that no permissions
// renders as [] rather than null, since PermissionsToJSONOutput preallocates a
// zero-length slice. A null here would break consumers that iterate the field.
func TestPermissionsToJSONOutput_EmptyInputIsEmptyArray(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(PermissionsToJSONOutput(nil))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != `[]` {
		t.Errorf("Marshal = %s, want []", got)
	}
}
