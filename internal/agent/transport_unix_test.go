//go:build !windows

package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func testEndpoint(t *testing.T) string {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "agent.sock")
}
func TestRejectExposedDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if l, err := Listen(filepath.Join(dir, "agent.sock")); err == nil {
		l.Close()
		t.Fatal("accepted exposed directory")
	}
}
