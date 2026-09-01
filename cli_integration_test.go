//go:build integration

package main_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/passbolt/go-passbolt-cli/internal/cmd"
	"github.com/passbolt/go-passbolt-cli/internal/testenv"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/tobischo/gokeepasslib/v3"
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

// TestCLI boots an ephemeral Passbolt via testcontainers, registers ada and
// admin users, materialises CLI TOML configs for each, and runs every .txtar
// scenario under testdata/scripts against the live server. Requires Docker.
func TestCLI(t *testing.T) {
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

	testscript.Run(t, testscript.Params{
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
			"kdbxcheck":  cmdKdbxCheck,
		},
		Condition: func(cond string) (bool, error) {
			if name, ok := strings.CutPrefix(cond, "env:"); ok {
				return os.Getenv(name) != "", nil
			}
			return false, fmt.Errorf("unknown condition %q", cond)
		},
	})
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

// kdbxcheck <file> <password> <group/.../entryTitle> [key=value]...
//
// Asserts that the KeePass file contains an entry at the given path: a
// '/'-separated chain of group names starting at the top-level group, with
// the final segment being the entry title. Same-named sibling groups (legal
// in Passbolt and KDBX) are all searched. Extra key=value args assert entry
// field contents (protected fields are unlocked first). `! kdbxcheck`
// asserts absence of the entry and takes no field args.
func cmdKdbxCheck(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) < 3 || (neg && len(args) > 3) {
		ts.Fatalf("usage: kdbxcheck <file> <password> <group/.../entryTitle> [key=value]...")
	}
	file, err := os.Open(ts.MkAbs(args[0]))
	if err != nil {
		ts.Fatalf("kdbxcheck open: %v", err)
	}
	defer file.Close()

	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(args[1])
	if err := gokeepasslib.NewDecoder(file).Decode(db); err != nil {
		ts.Fatalf("kdbxcheck decode: %v", err)
	}
	if err := db.UnlockProtectedEntries(); err != nil {
		ts.Fatalf("kdbxcheck unlock: %v", err)
	}

	segments := strings.Split(args[2], "/")
	if len(segments) < 2 {
		ts.Fatalf("kdbxcheck: path %q needs at least <topGroup>/<entryTitle>", args[2])
	}
	var found *gokeepasslib.Entry
	for i := range db.Content.Root.Groups {
		if entry := kdbxFindEntry(&db.Content.Root.Groups[i], segments); entry != nil {
			found = entry
			break
		}
	}
	switch {
	case found != nil && neg:
		ts.Fatalf("kdbxcheck: entry %q present, expected absent", args[2])
	case found == nil && !neg:
		ts.Fatalf("kdbxcheck: entry %q not found", args[2])
	case found == nil:
		return
	}

	for _, assertion := range args[3:] {
		key, want, ok := strings.Cut(assertion, "=")
		if !ok {
			ts.Fatalf("kdbxcheck: invalid field assertion %q (need key=value)", assertion)
		}
		if got := found.GetContent(key); got != want {
			ts.Fatalf("kdbxcheck: entry %q field %s = %q, want %q", args[2], key, got, want)
		}
	}
}

// kdbxFindEntry returns the first entry titled segments[len-1] under the
// group chain segments[1:len-1], or nil; group must match segments[0].
func kdbxFindEntry(group *gokeepasslib.Group, segments []string) *gokeepasslib.Entry {
	if group.Name != segments[0] {
		return nil
	}
	rest := segments[1:]
	if len(rest) == 1 {
		for i := range group.Entries {
			if group.Entries[i].GetTitle() == rest[0] {
				return &group.Entries[i]
			}
		}
		return nil
	}
	for i := range group.Groups {
		if entry := kdbxFindEntry(&group.Groups[i], rest); entry != nil {
			return entry
		}
	}
	return nil
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
