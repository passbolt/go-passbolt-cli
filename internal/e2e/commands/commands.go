// Package commands defines the custom commands e2e scripts can call.
package commands

import (
	"os"
	"runtime"
	"strconv"
	"uuid"

	"github.com/rogpeppe/go-internal/testscript"
)

// All returns the commands by name, for testscript.Params.Cmds.
func All() map[string]func(ts *testscript.TestScript, neg bool, args []string) {
	return map[string]func(ts *testscript.TestScript, neg bool, args []string){
		"jsoneq":     cmdJSONEq,
		"jsonget":    cmdJSONGet,
		"jsonexists": cmdJSONExists,
		"uuid":       cmdUUID,
		"payload":    cmdPayload,
		"defer":      cmdDefer,
		"perm":       cmdPerm,
	}
}

// jsoneq <file> <path> <expected> asserts the JSON value at <path> equals <expected>.
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

// jsonget <file> <path> <var> stores the JSON value at <path> in <var>, or "" if missing.
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

// jsonexists <file> <path> asserts the JSON value at <path> is not empty.
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

// uuid <var> stores a new UUIDv4 in <var>.
func cmdUUID(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("uuid does not support negation")
	}
	if len(args) != 1 {
		ts.Fatalf("usage: uuid <varname>")
	}
	ts.Setenv(args[0], uuid.New().String())
}

// defer <cmd> [args...] runs <cmd> when the script ends, last first, for cleanup.
// Errors are ignored: the target may already be gone.
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

// perm <file> <octal> asserts the file's permission bits. No-op on Windows.
func cmdPerm(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: perm <file> <octal>")
	}
	if runtime.GOOS == "windows" {
		return
	}
	want, err := strconv.ParseUint(args[1], 8, 32)
	if err != nil {
		ts.Fatalf("perm: bad mode %q: %v", args[1], err)
	}
	info, err := os.Lstat(ts.MkAbs(args[0]))
	if err != nil {
		ts.Fatalf("perm: %v", err)
	}
	got := info.Mode().Perm()
	if neg {
		if got == os.FileMode(want) {
			ts.Fatalf("perm: %s is %04o, want it not to be", args[0], got)
		}
		return
	}
	if got != os.FileMode(want) {
		ts.Fatalf("perm: %s is %04o, want %04o", args[0], got, want)
	}
}

// payload <var> <go-quoted-string> stores the unquoted string in <var>, so
// scripts can pass control bytes such as ESC.
func cmdPayload(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("payload does not support negation")
	}
	if len(args) != 2 {
		ts.Fatalf("usage: payload <varname> <quoted-string>")
	}
	s, err := strconv.Unquote(args[1])
	if err != nil {
		ts.Fatalf("payload: unquote %s: %v", args[1], err)
	}
	ts.Setenv(args[0], s)
}
