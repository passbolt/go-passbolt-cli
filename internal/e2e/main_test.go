//go:build integration

// Package e2e runs the .txtar scenarios against an ephemeral
// Passbolt. Requires Docker.
package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt-cli/internal/cmd"
	"github.com/passbolt/go-passbolt-cli/internal/e2e/commands"
	"github.com/passbolt/go-passbolt-cli/internal/e2e/server"
	"github.com/passbolt/go-passbolt/api"
	"github.com/rogpeppe/go-internal/testscript"
)

// TestMain registers the CLI as script commands: `pb` runs as ada, `pba` as admin.
func TestMain(m *testing.M) {
	wrap := func(cfgEnv string) func() {
		return func() {
			extra := []string{"--config", os.Getenv(cfgEnv), "--tlsSkipVerify"}
			os.Args = append([]string{"passbolt"}, append(extra, os.Args[1:]...)...)
			cmd.Execute()
		}
	}
	testscript.Main(m, map[string]func(){
		"passbolt": func() { cmd.Execute() },
		"pb":       wrap("CONFIG"),
		"pba":      wrap("CONFIG_ADMIN"),
	})
}

// cliParams boots Passbolt, creates the test users and returns the params
// shared by TestCLI and BenchmarkCLI.
func cliParams(t testing.TB) testscript.Params {
	ctx := t.Context()

	pb, err := server.Start(ctx)
	if err != nil {
		t.Fatalf("start passbolt: %v", err)
	}
	t.Cleanup(func() { _ = pb.Close(context.Background()) })

	// Passwords equal emails. adele is a third user to share with.
	admin, err := pb.CreateUser(ctx, "admin@passbolt.com", "Admin", "Tester", "admin", "admin@passbolt.com")
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := pb.EnableV5Resources(ctx, admin); err != nil {
		t.Fatalf("enable v5: %v", err)
	}
	ada, err := pb.CreateUser(ctx, "ada@passbolt.com", "Ada", "Lovelace", "user", "ada@passbolt.com")
	if err != nil {
		t.Fatalf("create ada user: %v", err)
	}
	if _, err := pb.CreateUser(ctx, "adele@passbolt.com", "Adele", "Goldberg", "user", "adele@passbolt.com"); err != nil {
		t.Fatalf("create adele user: %v", err)
	}

	// Scripts gate on these markers with [env:...].
	t.Setenv("HAS_ADMIN", "1")
	for _, slug := range advertisedResourceTypes(ctx, t, pb, admin) {
		t.Setenv(resourceTypeEnv(slug), "1")
	}

	return testscript.Params{
		Setup: func(env *testscript.Env) error {
			adaCfg := filepath.Join(env.WorkDir, "ada.toml")
			if err := os.WriteFile(adaCfg, []byte(tomlConfig(pb.BaseURL, ada)), 0600); err != nil {
				return err
			}
			env.Setenv("CONFIG", adaCfg)

			adminCfg := filepath.Join(env.WorkDir, "admin.toml")
			if err := os.WriteFile(adminCfg, []byte(tomlConfig(pb.BaseURL, admin)), 0600); err != nil {
				return err
			}
			env.Setenv("CONFIG_ADMIN", adminCfg)
			return nil
		},
		Cmds: commands.All(),
		Condition: func(cond string) (bool, error) {
			if name, ok := strings.CutPrefix(cond, "env:"); ok {
				return os.Getenv(name) != "", nil
			}
			return false, fmt.Errorf("unknown condition %q", cond)
		},
	}
}

// advertisedResourceTypes returns the resource type slugs the live server offers.
func advertisedResourceTypes(ctx context.Context, t testing.TB, pb *server.Passbolt, admin server.Credentials) []string {
	t.Helper()

	client, err := api.NewClient(nil, "go-passbolt-cli-tests", pb.BaseURL, admin.PrivateKey, admin.Password)
	if err != nil {
		t.Fatalf("new client for resource types: %v", err)
	}
	if err := client.Login(ctx); err != nil {
		t.Fatalf("login for resource types: %v", err)
	}
	defer func() { _ = client.Logout(ctx) }()

	types, err := client.GetResourceTypes(ctx, nil)
	if err != nil {
		t.Fatalf("getting resource types: %v", err)
	}

	slugs := make([]string, 0, len(types))
	for _, rType := range types {
		slugs = append(slugs, rType.Slug)
	}
	return slugs
}

// resourceTypeEnv maps v5-pin-code to HAS_TYPE_V5_PIN_CODE.
func resourceTypeEnv(slug string) string {
	return "HAS_TYPE_" + strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
}

// tomlConfig renders a CLI config file for c.
func tomlConfig(serverAddress string, c server.Credentials) string {
	return fmt.Sprintf(`serverAddress = %q
userPassword = %q
userPrivateKey = '''
%s
'''
`, serverAddress, c.Password, strings.TrimSpace(c.PrivateKey))
}
