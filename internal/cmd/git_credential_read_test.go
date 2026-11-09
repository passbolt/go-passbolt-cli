package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/viper"
)

func TestGitCredentialDecryptsResourceAndProtectsProtocol(t *testing.T) {
	const resourceID = "01234567-89ab-cdef-0123-456789abcdef"
	const typeID = "22222222-2222-2222-2222-222222222222"
	key, err := crypto.PGP().KeyGeneration().AddUserId("Credential test", "test@example.invalid").New().GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	defer key.ClearPrivateParams()
	armor, err := key.Armor()
	if err != nil {
		t.Fatal(err)
	}
	var encrypted, username atomic.Value
	var fail atomic.Bool
	respond := func(w http.ResponseWriter, body any) {
		json.NewEncoder(w).Encode(map[string]any{"header": map[string]any{"status": "success", "code": 200}, "body": body})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			json.NewEncoder(w).Encode(map[string]any{"header": map[string]any{"status": "error", "code": 403, "message": "private-error-marker"}, "body": "private-error-marker"})
			return
		}
		switch r.URL.Path {
		case "/resources/" + resourceID + ".json":
			respond(w, map[string]any{"id": resourceID, "resource_type_id": typeID, "name": "API token", "username": username.Load()})
		case "/resource-types/" + typeID + ".json":
			respond(w, map[string]any{"id": typeID, "slug": "password-and-description"})
		case "/secrets/resource/" + resourceID + ".json":
			respond(w, map[string]any{"resource_id": resourceID, "data": encrypted.Load()})
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := api.NewClient(server.Client(), "", server.URL, armor, "")
	if err != nil {
		t.Fatal(err)
	}
	oldProvider, oldRelease, oldData := util.ClientProvider, util.ClientRelease, util.DataOut
	oldOut, oldErr := util.Out, util.Err
	defer func() {
		util.ClientProvider = oldProvider
		util.ClientRelease = oldRelease
		util.DataOut = oldData
		util.Out = oldOut
		util.Err = oldErr
		viper.Reset()
		newCommandTree()
	}()
	util.ClientProvider = func(context.Context) (*api.Client, error) {
		util.Println("authentication progress")
		util.Eprintln("private-error-marker")
		return client, nil
	}
	util.ClientRelease = func(context.Context, *api.Client) {
		util.Println("cleanup progress")
		util.Eprintln("private-error-marker")
	}
	for _, test := range []struct {
		name, password, user, requested, want string
		wantError, serverError                bool
	}{
		{name: "exact bytes", password: "a=☃\u200bb", user: "token", want: "username=token\npassword=a=☃\u200bb\n\n"},
		{name: "token only", password: "token", want: "password=token\n\n"},
		{name: "requested username", password: "token", requested: "user", want: "username=user\npassword=token\n\n"},
		{name: "username mismatch", password: "token", user: "one", requested: "two", wantError: true},
		{name: "newline password", password: "token\npassword=injected", wantError: true},
		{name: "empty password", wantError: true},
		{name: "server error", password: "token", wantError: true, serverError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, _ := json.Marshal(map[string]string{"password": test.password, "description": ""})
			secret, err := client.EncryptMessage(string(data))
			if err != nil {
				t.Fatal(err)
			}
			encrypted.Store(secret)
			username.Store(test.user)
			fail.Store(test.serverError)
			viper.Reset()
			root := newCommandTree()
			viper.Set("timeout", "5s")
			var output, diagnostics bytes.Buffer
			util.DataOut = &output
			util.Out = &diagnostics
			util.Err = &diagnostics
			root.SetOut(&diagnostics)
			root.SetErr(&diagnostics)
			input := "protocol=https\nhost=api.hetzner.cloud\npath=" + resourceID + "\n"
			if test.requested != "" {
				input += "username=" + test.requested + "\n"
			}
			root.SetIn(strings.NewReader(input + "\n"))
			root.SetArgs([]string{"git-credential", "get"})
			err = root.Execute()
			if (err != nil) != test.wantError {
				t.Fatalf("unexpected result: %v", err)
			}
			if output.String() != test.want {
				t.Fatalf("protocol output %q want %q", output.String(), test.want)
			}
			if diagnostics.Len() != 0 {
				t.Fatalf("lookup leaked diagnostic text: %q", diagnostics.String())
			}
			if err != nil && strings.Contains(err.Error(), "private-error-marker") {
				t.Fatal("error leaked server response")
			}
		})
	}
}
