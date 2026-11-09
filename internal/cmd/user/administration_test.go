package user

import (
	"context"
	"encoding/json"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/passbolt/go-passbolt/api"
)

func TestUserAdministrationJSONPreservesUnknownFalseAndSCIM(t *testing.T) {
	for _, known := range []bool{false, true} {
		u := api.User{ID: "user-id", ScimEntry: json.RawMessage(`{"external_id":"directory-id","counter":9007199254740993}`)}
		if known {
			value := false
			u.IsMFAEnabled = &value
		}
		out := captureOutput(t, func() {
			if err := printUserDetail(u, true); err != nil {
				t.Fatal(err)
			}
		})
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &fields); err != nil {
			t.Fatal(err)
		}
		if known {
			if string(fields["is_mfa_enabled"]) != "false" {
				t.Errorf("lost explicit false: %s", out)
			}
		} else if _, ok := fields["is_mfa_enabled"]; ok {
			t.Errorf("invented MFA state: %s", out)
		}
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(fields["scim_entry"], &entry); err != nil {
			t.Fatal(err)
		}
		if string(entry["counter"]) != "9007199254740993" {
			t.Errorf("changed directory entry: %s", out)
		}
	}
}

func TestUserMFAListFlags(t *testing.T) {
	for _, args := range [][]string{{}, {"--mfa-enabled=false"}, {"--include-mfa"}, {"--column", "is_mfa_enabled"}, {"--scim-entry"}} {
		c := newTestUserListCmd()
		if err := c.ParseFlags(args); err != nil {
			t.Fatal(err)
		}
		cfg, err := parseUserListFlags(c)
		if err != nil {
			t.Fatal(err)
		}
		if len(args) == 0 {
			if cfg.includeMFA || cfg.mfaEnabled != nil || cfg.scimEntry {
				t.Fatal("ordinary list opted into admin fields")
			}
			continue
		}
		switch args[0] {
		case "--mfa-enabled=false":
			if cfg.mfaEnabled == nil || *cfg.mfaEnabled {
				t.Fatal("lost explicit false filter")
			}
		case "--include-mfa", "--column":
			if !cfg.includeMFA {
				t.Fatal("did not request MFA column data")
			}
		case "--scim-entry":
			if !cfg.scimEntry {
				t.Fatal("missing SCIM contain")
			}
		}
	}
}

func TestUserAdministrationCommandsReachExpectedEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name              string
		get               bool
		args              []string
		wantMFA, wantSCIM bool
		filter            string
	}{
		{name: "default-list", args: []string{"--json"}},
		{name: "false-filter", args: []string{"--json", "--mfa-enabled=false"}, wantMFA: true, filter: "false"},
		{name: "alias-column", args: []string{"--column", "IsMFAEnabled"}, wantMFA: true},
		{name: "scim-list", args: []string{"--json", "--scim-entry"}, wantSCIM: true},
		{name: "scim-get", get: true, args: []string{"--id", "11111111-1111-1111-1111-111111111111", "--json", "--scim-entry"}, wantSCIM: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				wantPath := "/users.json"
				if tc.get {
					wantPath = "/users/11111111-1111-1111-1111-111111111111.json"
				}
				if r.Method != "GET" || r.URL.Path != wantPath {
					t.Errorf("request=%s %s", r.Method, r.URL.Path)
				}
				q := r.URL.Query()
				if (q.Get("contain[is_mfa_enabled]") == "true") != tc.wantMFA || (q.Get("contain[scim_entry]") == "true") != tc.wantSCIM || q.Get("filter[is-mfa-enabled]") != tc.filter {
					t.Errorf("query=%s", r.URL.RawQuery)
				}
				user := map[string]any{"id": "11111111-1111-1111-1111-111111111111", "username": "ada@example.com", "is_mfa_enabled": false}
				var body any = user
				if !tc.get {
					body = []any{user}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"header": map[string]any{"status": "success", "code": 200}, "body": body})
			}))
			defer srv.Close()
			client, err := api.NewClient(srv.Client(), "test", srv.URL, "", "")
			if err != nil {
				t.Fatal(err)
			}
			runner := func(cmd *cobra.Command, fn func(context.Context, *api.Client) error) error {
				return fn(context.Background(), client)
			}
			c := newTestUserListCmd()
			if tc.get {
				c = &cobra.Command{Use: "user"}
				c.Flags().String("id", "", "")
				c.Flags().Bool("json", false, "")
				c.Flags().Bool("scim-entry", false, "")
				c.RunE = func(cmd *cobra.Command, args []string) error { return userGetWithClient(cmd, args, runner) }
			} else {
				c.RunE = func(cmd *cobra.Command, args []string) error { return userListWithClient(cmd, args, runner) }
			}
			c.SetArgs(tc.args)
			captureOutput(t, func() {
				if err := c.Execute(); err != nil {
					t.Fatal(err)
				}
			})
			if calls != 1 {
				t.Errorf("calls=%d", calls)
			}
		})
	}
}

func TestUserMFAFilterDistinguishesUnknownAndFalse(t *testing.T) {
	disabled := false
	users := []api.User{{ID: "unknown"}, {ID: "disabled", IsMFAEnabled: &disabled}}
	for _, tc := range []struct{ expression, id string }{{"is_mfa_enabled == false", "disabled"}, {"IsMFAEnabled == null", "unknown"}} {
		got, err := userColumns.Filter(context.Background(), users, tc.expression)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != tc.id {
			t.Errorf("%s returned %+v", tc.expression, got)
		}
	}
}

func TestSparseUserOptionalColumns(t *testing.T) {
	user := api.User{ID: "sparse"}
	for _, column := range []string{"first_name", "last_name", "role", "created_timestamp", "modified_timestamp"} {
		value, ok := userColumns.TableValue(user, column)
		if !ok || value != "" {
			t.Errorf("%s=%q supported=%t", column, value, ok)
		}
	}
	users, err := userColumns.Filter(context.Background(), []api.User{user}, "first_name == null && role == null && created_timestamp == null && modified_timestamp == null")
	if err != nil || len(users) != 1 {
		t.Errorf("users=%v error=%v", users, err)
	}
}
