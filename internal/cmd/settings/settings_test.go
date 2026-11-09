package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

const settingsUUID = "11111111-1111-1111-1111-111111111111"

func TestSettingsCommandsSendRequestsAndPreserveJSON(t *testing.T) {
	for _, tc := range []struct {
		op, method, plugin, path, input string
		args                            []string
		wantOutput                      bool
	}{
		{"get", "GET", "email-notifications", "/settings/emails/notifications.json", "", nil, true},
		{"get", "GET", "sso", "/sso/settings.json", "", []string{"--json"}, true},
		{"update", "POST", "smtp", "/smtp/settings.json", `{"enabled":false,"unset":null,"counter":9007199254740993}`, []string{"--input-file", "-", "--json"}, true},
		{"update", "POST", "locale", "/locale/settings.json", `{"locale":"de-DE"}`, []string{"--input-file", "-"}, false},
		{"delete", "DELETE", "directory-sync", "/directorysync/settings.json", "", nil, false},
		{"delete", "DELETE", "scim", "/scim/settings/" + settingsUUID + ".json", "", []string{"--id", settingsUUID}, false},
	} {
		t.Run(tc.op+"/"+tc.plugin, func(t *testing.T) {
			requests, auth := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("request=%s %s", r.Method, r.URL.Path)
				}
				if tc.input != "" {
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if tc.plugin == "smtp" && (string(body["enabled"]) != "false" || string(body["unset"]) != "null" || string(body["counter"]) != "9007199254740993") {
						t.Errorf("modified input: %v", body)
					}
					if tc.plugin == "locale" && string(body["locale"]) != `"de-DE"` {
						t.Errorf("body=%v", body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"header":{"status":"success","code":200},"body":{"enabled":false,"counter":9007199254740993,"sources_file":{"x":true}}}`))
			}))
			defer srv.Close()
			client, err := api.NewClient(srv.Client(), "test", srv.URL, "", "")
			if err != nil {
				t.Fatal(err)
			}
			c := newSettingsCommand(tc.op, func(cmd *cobra.Command, fn func(context.Context, *api.Client) error) error {
				auth++
				return fn(context.Background(), client)
			})
			var out bytes.Buffer
			c.SetOut(&out)
			c.SetErr(&out)
			c.SetIn(strings.NewReader(tc.input))
			c.SetArgs(append([]string{"--plugin", tc.plugin}, tc.args...))
			if err := c.Execute(); err != nil {
				t.Fatal(err)
			}
			if requests != 1 || auth != 1 {
				t.Fatalf("requests=%d auth=%d", requests, auth)
			}
			if !tc.wantOutput {
				if out.Len() != 0 {
					t.Errorf("unexpected mutation output: %s", out.String())
				}
				return
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(out.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if string(body["enabled"]) != "false" || string(body["counter"]) != "9007199254740993" || body["sources_file"] == nil {
				t.Errorf("modified response: %s", out.String())
			}
		})
	}
}

func TestSettingsValidationHappensBeforeAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name, op, input string
		args            []string
	}{
		{"missing plugin", "get", "", nil},
		{"unknown plugin", "get", "", []string{"--plugin", "unknown"}},
		{"write-only plugin", "get", "", []string{"--plugin", "locale"}},
		{"read-only plugin", "update", "{}", []string{"--plugin", "export-policies", "--input-file", "-"}},
		{"unsupported delete", "delete", "", []string{"--plugin", "smtp"}},
		{"missing input", "update", "", []string{"--plugin", "smtp"}},
		{"missing file", "update", "", []string{"--plugin", "smtp", "--input-file", "/does-not-exist/settings.json"}},
		{"empty input", "update", "", []string{"--plugin", "smtp", "--input-file", "-"}},
		{"bad JSON", "update", "{bad}", []string{"--plugin", "smtp", "--input-file", "-"}},
		{"null input", "update", "null", []string{"--plugin", "smtp", "--input-file", "-"}},
		{"array input", "update", "[]", []string{"--plugin", "smtp", "--input-file", "-"}},
		{"scalar input", "update", "42", []string{"--plugin", "smtp", "--input-file", "-"}},
		{"multiple values", "update", "{} {}", []string{"--plugin", "smtp", "--input-file", "-"}},
		{"missing UUID", "delete", "", []string{"--plugin", "self-registration"}},
		{"bad UUID", "delete", "", []string{"--plugin", "sso", "--id", "../settings"}},
		{"irrelevant UUID", "delete", "", []string{"--plugin", "secret-revisions", "--id", settingsUUID}},
		{"positional args", "get", "", []string{"--plugin", "smtp", "extra"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			auth := false
			c := newSettingsCommand(tc.op, func(cmd *cobra.Command, fn func(context.Context, *api.Client) error) error {
				auth = true
				return fmt.Errorf("unexpected authentication")
			})
			var out bytes.Buffer
			c.SetOut(&out)
			c.SetErr(&out)
			c.SetIn(strings.NewReader(tc.input))
			c.SetArgs(tc.args)
			if err := c.Execute(); err == nil {
				t.Fatal("accepted invalid request")
			}
			if auth {
				t.Fatal("authenticated before rejecting invalid input")
			}
			if strings.Contains(out.String(), tc.input) && tc.input == "{bad}" {
				t.Fatal("echoed invalid settings input")
			}
		})
	}
}

func TestSettingsUpdateReadsFileAndPropagatesAPIError(t *testing.T) {
	input := filepath.Join(t.TempDir(), "smtp.json")
	if err := os.WriteFile(input, []byte(`{"sender":"ada@example.com"}`), 0600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/smtp/settings.json" {
			t.Errorf("request=%s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["sender"] != "ada@example.com" {
			t.Errorf("body=%v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"header":{"status":"error","code":403,"message":"forbidden"},"body":{}}`))
	}))
	defer srv.Close()
	client, err := api.NewClient(srv.Client(), "test", srv.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	c := newSettingsCommand("update", func(cmd *cobra.Command, fn func(context.Context, *api.Client) error) error {
		return fn(context.Background(), client)
	})
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"--plugin", "smtp", "--input-file", input})
	if err := c.Execute(); err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("error=%v", err)
	}
}

func TestSettingsPluginCompletionFiltersByOperation(t *testing.T) {
	for _, tc := range []struct{ op, included, excluded string }{{"get", "smtp", "locale"}, {"update", "locale", "export-policies"}, {"delete", "scim", "smtp"}} {
		c := newSettingsCommand(tc.op, nil)
		complete, ok := c.GetFlagCompletionFunc("plugin")
		if !ok {
			t.Fatalf("missing completion for %s", tc.op)
		}
		names, directive := complete(c, nil, "")
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive=%v", directive)
		}
		joined := " " + strings.Join(names, " ") + " "
		if !strings.Contains(joined, " "+tc.included+" ") || strings.Contains(joined, " "+tc.excluded+" ") {
			t.Errorf("completion for %s=%v", tc.op, names)
		}
	}
}

func TestSettingsCommandsWithInheritedParentFlags(t *testing.T) {
	for _, tc := range []struct {
		operation string
		args      []string
	}{
		{"get", []string{"--json", "organization-settings", "--plugin", "smtp"}},
		{"delete", []string{"--id", settingsUUID, "organization-settings", "--plugin", "scim"}},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			requests := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if tc.operation == "delete" && r.URL.Path != "/scim/settings/"+settingsUUID+".json" {
					t.Errorf("path=%s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"header":{"status":"success","code":200},"body":{"enabled":false}}`))
			}))
			defer srv.Close()
			client, err := api.NewClient(srv.Client(), "test", srv.URL, "", "")
			if err != nil {
				t.Fatal(err)
			}
			parent := &cobra.Command{Use: tc.operation}
			if tc.operation == "get" {
				parent.PersistentFlags().BoolP("json", "j", false, "")
			} else {
				parent.PersistentFlags().String("id", "", "")
			}
			child := newSettingsCommand(tc.operation, func(cmd *cobra.Command, fn func(context.Context, *api.Client) error) error {
				return fn(context.Background(), client)
			})
			parent.AddCommand(child)
			var out bytes.Buffer
			parent.SetOut(&out)
			parent.SetErr(&out)
			parent.SetArgs(tc.args)
			if err := parent.Execute(); err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests=%d", requests)
			}
			if tc.operation == "get" && !json.Valid(out.Bytes()) {
				t.Errorf("invalid JSON output %s", out.String())
			}
		})
	}
}
