package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/viper"
)

func TestCredentialValidationAndNoopDoNotAuthenticate(t *testing.T) {
	oldProvider := util.ClientProvider
	calls := 0
	util.ClientProvider = func(context.Context) (*api.Client, error) {
		calls++
		t.Fatal("unexpected authentication")
		return nil, nil
	}
	defer func() { util.ClientProvider = oldProvider; viper.Reset(); newCommandTree() }()
	for _, test := range []struct {
		op, input string
		wantError bool
	}{
		{op: "get", input: "protocol=https\nhost=example.com/invalid\npath=not-a-uuid\n\n", wantError: true},
		{op: "store", input: "password=do-not-store\n\n"},
		{op: "erase", input: "protocol=https\n\n"},
		{op: "unknown", wantError: true},
	} {
		viper.Reset()
		root := newCommandTree()
		viper.Set("agent", true)
		root.SetArgs([]string{"git-credential", test.op})
		root.SetIn(strings.NewReader(test.input))
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		err := root.Execute()
		if (err != nil) != test.wantError {
			t.Fatalf("%s: %v", test.op, err)
		}
		if out.Len() != 0 {
			t.Fatalf("%s wrote non-protocol output: %q", test.op, out.String())
		}
	}
	if calls != 0 {
		t.Fatal("authenticated invalid/read-only request")
	}
}
