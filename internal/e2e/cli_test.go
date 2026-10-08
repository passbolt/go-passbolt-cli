//go:build integration

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// TestCLI runs every scenario in scenarios/, one subtest per directory.
func TestCLI(t *testing.T) {
	p := cliParams(t)
	for _, dir := range scenarioDirs(t) {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			t.Parallel()
			dp := p
			dp.Dir = dir
			testscript.Run(t, dp)
		})
	}
}

// scenarioDirs returns the directories under scenarios, one per area.
func scenarioDirs(t testing.TB) []string {
	entries, err := os.ReadDir("scenarios")
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join("scenarios", e.Name()))
		}
	}
	return dirs
}
