//go:build integration

package main_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// BenchmarkCLI runs every .txtar scenario as a sub-benchmark, one row per script. Each `pb`
// line re-execs the CLI, so a row is real end-to-end cost and cannot be CPU-profiled.
func BenchmarkCLI(b *testing.B) {
	p := cliParams(b)
	files, err := filepath.Glob(filepath.Join(p.Dir, "*.txtar"))
	if err != nil {
		b.Fatal(err)
	}
	for _, f := range files {
		b.Run(strings.TrimSuffix(filepath.Base(f), ".txtar"), func(b *testing.B) {
			fp := p
			fp.Dir, fp.Files = "", []string{f}
			for b.Loop() {
				testscript.RunT(benchT{b, &strings.Builder{}}, fp)
			}
		})
	}
}

// benchT adapts *testing.B to testscript.T. Run executes the script inline so
// the whole run stays inside b.Loop, and Parallel is a no-op for the same
// reason. A script whose [env:...] condition is false calls Skip, which skips
// that sub-benchmark rather than failing it. testscript logs every script's
// transcript; that is held back and printed only if the script fails, so the
// benchmark output stays one row per scenario.
type benchT struct {
	*testing.B
	log *strings.Builder
}

func (t benchT) Run(_ string, f func(testscript.T)) { f(benchT{t.B, &strings.Builder{}}) }
func (benchT) Parallel()                            {}
func (benchT) Verbose() bool                        { return testing.Verbose() }
func (t benchT) Log(args ...any)                    { fmt.Fprintln(t.log, args...) }
func (t benchT) Fatal(args ...any)                  { t.B.Log(t.log.String()); t.B.Fatal(args...) }
func (t benchT) FailNow()                           { t.B.Log(t.log.String()); t.B.FailNow() }
