package main_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/passbolt/go-passbolt-cli/cmd"
	"github.com/rogpeppe/go-internal/testscript"
)

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

// TestCLI runs every .txtar under testdata/scripts as an independent UAT
// scenario against a live Passbolt instance. Skips cleanly when the test
// environment isn't configured.
func TestCLI(t *testing.T) {
	if os.Getenv("PASSBOLT_TEST_URL") == "" {
		t.Skip("PASSBOLT_TEST_URL not set, skipping CLI integration tests")
	}

	cfgPath := os.Getenv("PASSBOLT_TEST_CLI_CONFIG")
	if cfgPath == "" {
		cfgPath = filepath.Join(os.Getenv("HOME"), ".config/go-passbolt-cli-ada/go-passbolt-cli.toml")
	}
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Skipf("CLI config %s not readable: %v", cfgPath, err)
	}

	// Admin config is optional. Scripts that don't use it (most) skip the
	// `pba` Cmd entirely; scripts that do (group create) skip themselves
	// when this isn't available.
	adminPath := os.Getenv("PASSBOLT_TEST_CLI_ADMIN_CONFIG")
	if adminPath == "" {
		adminPath = filepath.Join(os.Getenv("HOME"), ".config/go-passbolt-cli-admin/go-passbolt-cli.toml")
	}
	adminData, _ := os.ReadFile(adminPath) // missing → empty, scripts gate via [env:HAS_ADMIN]
	if len(adminData) > 0 {
		// Process-level marker the Condition handler reads. The per-script
		// $CONFIG_ADMIN path is set in Setup; this just says "admin exists".
		t.Setenv("HAS_ADMIN", "1")
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts",
		Setup: func(env *testscript.Env) error {
			target := filepath.Join(env.WorkDir, "ada.toml")
			if err := os.WriteFile(target, cfgData, 0600); err != nil {
				return err
			}
			env.Setenv("CONFIG", target)
			if len(adminData) > 0 {
				adminTarget := filepath.Join(env.WorkDir, "admin.toml")
				if err := os.WriteFile(adminTarget, adminData, 0600); err != nil {
					return err
				}
				env.Setenv("CONFIG_ADMIN", adminTarget)
			}
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"jsoneq":     cmdJSONEq,
			"jsonget":    cmdJSONGet,
			"jsonexists": cmdJSONExists,
			"uuid":       cmdUUID,
			"defer":      cmdDefer,
		},
		// `[env:NAME]` is true when the named env var is set and non-empty
		// in the test process. Used to gate scripts that depend on
		// optional configuration (admin config for group ops).
		Condition: func(cond string) (bool, error) {
			if name, ok := strings.CutPrefix(cond, "env:"); ok {
				return os.Getenv(name) != "", nil
			}
			return false, fmt.Errorf("unknown condition %q", cond)
		},
	})
}

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

// uuid <varname>  — generate a fresh UUIDv4 into env var <varname>.
func cmdUUID(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("uuid does not support negation")
	}
	if len(args) != 1 {
		ts.Fatalf("usage: uuid <varname>")
	}
	ts.Setenv(args[0], uuid.NewString())
}

// jsonPath resolves a path against parsed JSON.
//
// Path grammar:
//
//	field            object field access
//	a.b.c            dotted nested access
//	[k=v]            filter: select first element of an array whose field k equals v
//	[k=v].field      filter then field access
//
// Missing fields and unmatched filters return ("", nil) — they are not errors;
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
