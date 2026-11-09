package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

func TestUserRecoverRequiresUsername(t *testing.T) {
	if err := UserRecoverCmd.ValidateRequiredFlags(); err == nil {
		t.Fatal("expected required username error")
	}
	c := &cobra.Command{}
	c.Flags().String("username", " \t", "")
	if err := UserRecover(c, nil); err == nil || !strings.Contains(err.Error(), "username") {
		t.Errorf("error = %v, want username validation", err)
	}
}

func TestRecoverUserRequestAndError(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/users/recover.json" {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["username"] != "ada@example.com" || len(body) != 1 {
					t.Errorf("body = %#v", body)
				}
				w.Header().Set("Content-Type", "application/json")
				if fail {
					w.Write([]byte(`{"header":{"status":"error","code":400,"message":"invalid user"},"body":{}}`))
				} else {
					w.Write([]byte(`{"header":{"status":"success","code":200},"body":null}`))
				}
			}))
			defer srv.Close()
			c, err := api.NewClient(srv.Client(), "test", srv.URL, "", "")
			if err != nil {
				t.Fatal(err)
			}
			err = recoverUser(context.Background(), c, "ada@example.com")
			if fail {
				if err == nil || !strings.Contains(err.Error(), "invalid user") {
					t.Errorf("error = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
