// Package posixmode holds the permission-bit helpers behind securefile on Unix.
//
// It has no build tag: it only reads mode bits, so its tests run everywhere.
package posixmode

import (
	"io/fs"
	"os"
)

// SecretMode is the mode for files holding credentials in cleartext.
const SecretMode os.FileMode = 0o600

// IsLoose reports whether anyone but the owner can reach the file.
func IsLoose(info fs.FileInfo) bool { return info.Mode().Perm()&0o077 != 0 }
