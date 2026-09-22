//go:build integration

// End-to-end CLI harness: runs the .txtar scenarios in internal/testdata
// against an ephemeral Passbolt booted by internal/testenv.
//
// Three parts, in order:
//   - the harness entry points and the ephemeral environment they boot;
//   - the custom commands the scripts call (jsoneq, jsonget, jsonexists,
//     uuid, defer);
//   - the JSON path resolver backing the three json* commands.
//
// cliParams is shared with BenchmarkCLI in cli_bench_test.go.

package main_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/passbolt/go-passbolt-cli/internal/cmd"
	"github.com/passbolt/go-passbolt-cli/internal/testenv"
	"github.com/passbolt/go-passbolt/api"
	"github.com/rogpeppe/go-internal/testscript"
)

// ---- Harness entry points ------------------------------------------------

// TestMain wires the test binary so that any `passbolt`, `pb`, or `pba`
// directive in a .txtar script re-execs this binary and dispatches to the
// real CLI. `pb` runs as the standard ada test user; `pba` runs as admin
// and is only used by scripts that exercise admin-only operations (group
// CRUD on this server). Both pre-apply --config and --tlsSkipVerify so
// scripts stay focused on the operation.
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

// TestCLI runs every .txtar scenario under internal/testdata against an ephemeral Passbolt.
// Requires Docker.
func TestCLI(t *testing.T) {
	testscript.Run(t, cliParams(t))
}

// ---- Test environment ----------------------------------------------------

// cliParams boots an ephemeral Passbolt, registers the users, sets the [env:...] markers and
// returns the testscript parameters TestCLI and BenchmarkCLI share. Requires Docker.
func cliParams(t testing.TB) testscript.Params {
	ctx := t.Context()

	pb, err := testenv.Start(ctx)
	if err != nil {
		t.Fatalf("start passbolt testenv: %v", err)
	}
	t.Cleanup(func() { _ = pb.Close(context.Background()) })

	// Email/password convention mirrors the existing Passbolt seed-data
	// fixtures (password == email) so test scripts that hard-code seeded
	// emails like "ada@passbolt.com" line up. adele is created so scripts
	// that share resources/folders with a third user (28, 31) find her.
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

	// Marker for scripts gated on admin availability via [env:HAS_ADMIN].
	t.Setenv("HAS_ADMIN", "1")

	// One marker per resource type this server advertises, so a script covering a type that
	// arrived in a later release can gate on it: v5-pin-code, for instance, only exists from
	// 5.12, and without the gate its script would fail on every older release.
	for _, slug := range advertisedResourceTypes(ctx, t, pb, admin) {
		t.Setenv(resourceTypeEnv(slug), "1")
	}

	return testscript.Params{
		Dir: "internal/testdata",
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
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"jsoneq":     cmdJSONEq,
			"jsonget":    cmdJSONGet,
			"jsonexists": cmdJSONExists,
			"uuid":       cmdUUID,
			"defer":      cmdDefer,
		},
		Condition: func(cond string) (bool, error) {
			if name, ok := strings.CutPrefix(cond, "env:"); ok {
				return os.Getenv(name) != "", nil
			}
			return false, fmt.Errorf("unknown condition %q", cond)
		},
	}
}

// advertisedResourceTypes returns the resource type slugs the live server offers.
func advertisedResourceTypes(ctx context.Context, t testing.TB, pb *testenv.Passbolt, admin testenv.Credentials) []string {
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

// resourceTypeEnv maps a slug to the marker scripts gate on: v5-pin-code becomes
// HAS_TYPE_V5_PIN_CODE, used as [env:HAS_TYPE_V5_PIN_CODE].
func resourceTypeEnv(slug string) string {
	return "HAS_TYPE_" + strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
}

// tomlConfig renders a CLI TOML config for the given credentials. The PGP
// armored private key is a multi-line string, so we use TOML triple-quoted
// literals; the password is short and safe for a bare key=value line.
func tomlConfig(serverAddress string, c testenv.Credentials) string {
	return fmt.Sprintf(`serverAddress = %q
userPassword = %q
userPrivateKey = '''
%s
'''
`, serverAddress, c.Password, strings.TrimSpace(c.PrivateKey))
}

// ---- Script commands -----------------------------------------------------

// jsoneq <file> <path> <expected>
//
// Asserts that the JSON value at <path> in <file> equals <expected>. When
// negated (`! jsoneq ...`) the assertion is inverted.
func cmdJSONEq(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 3 {
		ts.Fatalf("usage: jsoneq <file> <path> <expected>")
	}
	got, err := jsonPath(ts.ReadFile(args[0]), args[1])
	if err != nil {
		ts.Fatalf("jsoneq %s %s: %v", args[0], args[1], err)
	}
	eq := got == args[2]
	switch {
	case eq && neg:
		ts.Fatalf("jsoneq %s %s == %q, expected ≠", args[0], args[1], args[2])
	case !eq && !neg:
		ts.Fatalf("jsoneq %s %s = %q, want %q", args[0], args[1], got, args[2])
	}
}

// jsonget <file> <path> <varname>
//
// Captures the JSON value at <path> in <file> into env var <varname>. Sets
// empty string if the path doesn't resolve.
func cmdJSONGet(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("jsonget does not support negation")
	}
	if len(args) != 3 {
		ts.Fatalf("usage: jsonget <file> <path> <varname>")
	}
	got, err := jsonPath(ts.ReadFile(args[0]), args[1])
	if err != nil {
		ts.Fatalf("jsonget %s %s: %v", args[0], args[1], err)
	}
	ts.Setenv(args[2], got)
}

// jsonexists <file> <path>
//
// Succeeds if the JSON path resolves to a non-empty value. Use `! jsonexists`
// to assert absence.
func cmdJSONExists(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: jsonexists <file> <path>")
	}
	got, err := jsonPath(ts.ReadFile(args[0]), args[1])
	if err != nil {
		ts.Fatalf("jsonexists %s %s: %v", args[0], args[1], err)
	}
	exists := got != ""
	switch {
	case exists && neg:
		ts.Fatalf("jsonexists %s %s = %q, expected absent", args[0], args[1], got)
	case !exists && !neg:
		ts.Fatalf("jsonexists %s %s is empty, expected present", args[0], args[1])
	}
}

// uuid <varname>: generate a fresh UUIDv4 into env var <varname>.
func cmdUUID(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("uuid does not support negation")
	}
	if len(args) != 1 {
		ts.Fatalf("usage: uuid <varname>")
	}
	ts.Setenv(args[0], uuid.New().String())
}

// defer <cmd> [args...]
//
// Schedules <cmd> to run at end-of-script (LIFO order, mirroring Go's defer)
// even if a later assertion fails. Used for resource cleanup so failures
// don't leak state on the live server. Errors from the deferred command are
// intentionally swallowed: the resource may already be gone (e.g. the script
// deleted it explicitly) or never created (script failed early), and the
// failure that mattered has already been reported.
func cmdDefer(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("defer does not support negation")
	}
	if len(args) < 1 {
		ts.Fatalf("usage: defer <cmd> [args...]")
	}
	name, rest := args[0], args[1:]
	ts.Defer(func() {
		_ = ts.Exec(name, rest...)
	})
}

// ---- JSON path -----------------------------------------------------------

// jsonPath resolves a path against parsed JSON.
//
// Path grammar:
//
//	field            object field access
//	a.b.c            dotted nested access
//	[k=v]            filter: select first element of an array whose field k equals v
//	[k=v].field      filter then field access
//
// Missing fields and unmatched filters return ("", nil). They are not errors;
// callers distinguish via jsoneq vs jsonexists.
func jsonPath(data, path string) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return "", fmt.Errorf("invalid JSON: %v", err)
	}
	cur := v
	for _, seg := range tokenize(path) {
		if cur == nil {
			return "", nil
		}
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			arr, ok := cur.([]any)
			if !ok {
				return "", fmt.Errorf("filter %q applied to non-array %T", seg, cur)
			}
			body := seg[1 : len(seg)-1]
			eq := strings.SplitN(body, "=", 2)
			if len(eq) != 2 {
				return "", fmt.Errorf("invalid filter %q (need [key=value])", seg)
			}
			key, want := eq[0], eq[1]
			cur = nil
			for _, el := range arr {
				m, ok := el.(map[string]any)
				if !ok {
					continue
				}
				if stringify(m[key]) == want {
					cur = m
					break
				}
			}
			continue
		}
		m, ok := cur.(map[string]any)
		if !ok {
			return "", fmt.Errorf("field %q applied to non-object %T", seg, cur)
		}
		cur = m[seg]
	}
	return stringify(cur), nil
}

// tokenize splits a path like "a.b[k=v].c" into ["a", "b", "[k=v]", "c"].
func tokenize(path string) []string {
	if path == "" {
		return nil
	}
	var out []string
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			out = append(out, buf.String())
			buf.Reset()
		}
	}
	i := 0
	for i < len(path) {
		switch c := path[i]; c {
		case '.':
			flush()
			i++
		case '[':
			flush()
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				out = append(out, path[i:])
				return out
			}
			out = append(out, path[i:i+end+1])
			i += end + 1
		default:
			buf.WriteByte(c)
			i++
		}
	}
	flush()
	return out
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}
