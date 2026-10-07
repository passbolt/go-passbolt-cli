//go:build integration

package e2e

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// BenchmarkCLI runs each scenario as a sub-benchmark. `pb` re-execs the CLI,
// so rows measure end-to-end cost.
func BenchmarkCLI(b *testing.B) {
	p := cliParams(b)
	files, err := filepath.Glob("scenarios/*/*.txtar")
	if err != nil {
		b.Fatal(err)
	}
	for _, f := range files {
		name := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(f), "scenarios/"), ".txtar")
		b.Run(name, func(b *testing.B) {
			fp := p
			fp.Files = []string{f}
			for b.Loop() {
				testscript.RunT(benchT{b, &strings.Builder{}}, fp)
			}
		})
	}
}

// benchT adapts *testing.B to testscript.T. Scripts run inline, not in
// parallel, and their log is printed only on failure.
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
