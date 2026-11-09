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

func TestUpdateUserStateSendsRequestedTransition(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enable", true: "disable"}[disabled], func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" || r.URL.Path != "/users/11111111-1111-1111-1111-111111111111.json" {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				value, ok := body["disabled"]
				if !ok || len(body) != 1 {
					t.Errorf("body = %#v", body)
				}
				if disabled {
					if _, ok := value.(string); !ok {
						t.Errorf("disable value = %v", value)
					}
				} else if value != nil {
					t.Errorf("enable value = %v", value)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"header":{"status":"success","code":200},"body":{}}`))
			}))
			defer srv.Close()
			c, err := api.NewClient(srv.Client(), "test", srv.URL, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if err := updateUserState(context.Background(), c, "11111111-1111-1111-1111-111111111111", disabled); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUserUpdateRejectsMixedStateAndProfileFlags(t *testing.T) {
	for _, flag := range []string{"firstname", "lastname", "role"} {
		t.Run(flag, func(t *testing.T) {
			c := &cobra.Command{}
			c.Flags().String("id", "11111111-1111-1111-1111-111111111111", "")
			for _, f := range []string{"firstname", "lastname", "role"} {
				c.Flags().String(f, "", "")
			}
			c.Flags().Bool("disabled", false, "")
			if err := c.ParseFlags([]string{"--disabled=false", "--" + flag, "value"}); err != nil {
				t.Fatal(err)
			}
			if err := UserUpdate(c, nil); err == nil || !strings.Contains(err.Error(), "cannot combine") {
				t.Errorf("error = %v, want mixed flag error before authentication", err)
			}
		})
	}
}

func TestUserUpdateRegistersDisabledFlag(t *testing.T) {
	if UserUpdateCmd.Flags().Lookup("disabled") == nil {
		t.Fatal("missing --disabled flag")
	}
}
