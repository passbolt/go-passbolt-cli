package posixmode

import (
	"io/fs"
	"os"
	"testing"
	"time"
)

// modeInfo is a FileInfo carrying only a mode.
type modeInfo os.FileMode

func (m modeInfo) Name() string       { return "f" }
func (m modeInfo) Size() int64        { return 0 }
func (m modeInfo) Mode() fs.FileMode  { return fs.FileMode(m) }
func (m modeInfo) ModTime() time.Time { return time.Time{} }
func (m modeInfo) IsDir() bool        { return fs.FileMode(m).IsDir() }
func (m modeInfo) Sys() any           { return nil }

func TestIsLoose(t *testing.T) {
	for _, tt := range []struct {
		mode os.FileMode
		want bool
	}{
		{SecretMode, false},
		{0o400, false},
		{0o700 | fs.ModeDir, false},
		{0o640, true},
		{0o604, true},
		{0o707, true},
		{0o777 | fs.ModeDir, true},
	} {
		if got := IsLoose(modeInfo(tt.mode)); got != tt.want {
			t.Errorf("IsLoose(%v) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}
