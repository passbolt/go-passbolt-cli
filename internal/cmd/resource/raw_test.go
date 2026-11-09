package resource

import (
	"context"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestRawFlagValidationBeforeAuthentication(t *testing.T) {
	defer ResetCommands()
	previous := util.ClientProvider
	util.ClientProvider = func(context.Context) (*api.Client, error) {
		t.Fatal("invalid raw flags authenticated")
		return nil, nil
	}
	defer func() { util.ClientProvider = previous; viper.Reset() }()
	for _, args := range [][]string{{"--raw", "--json"}, {"--field", "password"}, {"--raw", "--field", ""}, {"--raw", "--debug"}} {
		ResetCommands()
		root := &cobra.Command{Use: "get", SilenceErrors: true, SilenceUsage: true}
		root.PersistentFlags().Bool("json", false, "")
		root.PersistentFlags().Bool("debug", false, "")
		root.AddCommand(ResourceGetCmd)
		viper.Reset()
		viper.BindPFlag("debug", root.PersistentFlags().Lookup("debug"))
		root.SetArgs(append([]string{"resource", "--id", "01234567-89ab-cdef-0123-456789abcdef"}, args...))
		if err := root.Execute(); err == nil {
			t.Fatalf("accepted flags %v", args)
		}
	}
}
func TestRawFieldExtraction(t *testing.T) {
	metadata := map[string]any{"username": "alice", "description": "public"}
	secret := map[string]any{"password": " ☃\u200bsecret\n", "description": "private", "totp": map[string]any{"key": "secret"}}
	for _, test := range []struct{ field, want string }{{"password", " ☃\u200bsecret\n"}, {"username", "alice"}, {"description", "private"}} {
		got, err := extractRawField(metadata, secret, test.field)
		if err != nil || got != test.want {
			t.Fatalf("%s: %q %v", test.field, got, err)
		}
	}
	for _, field := range []string{"missing", "totp"} {
		if _, err := extractRawField(metadata, secret, field); err == nil || strings.Contains(err.Error(), "private") {
			t.Fatal("invalid field extraction")
		}
	}
}
