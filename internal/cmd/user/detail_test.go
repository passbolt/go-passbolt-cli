package user

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/passbolt/go-passbolt/api"
)

func TestUserDetailOutputIncludesState(t *testing.T) {
	disabled := &api.Time{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	u := api.User{ID: "user-id", Username: "ada@example.com", Active: true, Deleted: true, Disabled: disabled,
		Profile: &api.Profile{FirstName: "Ada", LastName: "Lovelace"}, Role: &api.Role{Name: "admin"}, Created: disabled, Modified: disabled}
	out := captureOutput(t, func() {
		if err := printUserDetail(u, true); err != nil {
			t.Fatal(err)
		}
	})
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{"id": "user-id", "username": "ada@example.com", "first_name": "Ada", "last_name": "Lovelace", "role": "admin", "active": true, "deleted": true, "disabled": true, "created_timestamp": "2026-01-02T03:04:05Z", "modified_timestamp": "2026-01-02T03:04:05Z"} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v", key, got[key], want)
		}
	}
}

func TestUserDetailOutputHandlesMissingOptionalData(t *testing.T) {
	out := captureOutput(t, func() {
		if err := printUserDetail(api.User{ID: "user-id"}, true); err != nil {
			t.Fatal(err)
		}
	})
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"active", "deleted", "disabled"} {
		if got[key] != false {
			t.Errorf("%s = %v, want false", key, got[key])
		}
	}
	for _, key := range []string{"first_name", "last_name", "role", "created_timestamp", "modified_timestamp"} {
		if _, ok := got[key]; ok {
			t.Errorf("unexpected %s in sparse output", key)
		}
	}
}
