package securefile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// The platform contract: securefile_unix.go and securefile_windows.go each implement exactly
// these, with their helpers in posixmode and winacl. A missing or mistyped one fails the build
// on that platform; make vet builds both.
var (
	_ func(string) (*os.File, error)    = openSecret
	_ func(*os.File) bool               = ownedByCaller
	_ func(*os.File, fs.FileInfo) error = tightenIfLoose
	_ func(string, os.FileMode) error   = repairMode
	_ func(string) (string, bool)       = describeLoose
	_ func(string) string               = fixCommand
	_ bool                              = hasPOSIXModes
)

// Create creates or truncates path as a regular file owned by the caller, mode 0600 or its Windows ACL equivalent.
func Create(path string) (_ *os.File, err error) {
	// No truncation yet: a refusal below must leave the file as it was.
	file, err := openSecret(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = file.Close()
		}
	}()

	// Checked on the open descriptor, not on the path: nothing can be swapped in between.
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || !ownedByCaller(file) {
		return nil, fmt.Errorf("refusing to write %q: not a regular file you own", path)
	}
	// Creation alone secures only a new file, so an existing loose one is tightened here.
	if err = tightenIfLoose(file, info); err != nil {
		return nil, fmt.Errorf("securing %q: %w", path, err)
	}
	if err = file.Truncate(0); err != nil {
		return nil, err
	}
	return file, nil
}

// errPermissionsIgnored is a tightening that was accepted but did not take effect.
var errPermissionsIgnored = errors.New("permissions did not change, so others can still read it")

// RepairMode tightens path to mode when anyone but the owner can reach it; a missing path
// is fine. Only the default config path and its directory reach this.
func RepairMode(path string, mode os.FileMode) error { return repairMode(path, mode) }

// Exposure reports whether others can reach path, how, and the command that fixes it,
// without touching it. A missing path is not exposed.
func Exposure(path string) (detail, fix string, exposed bool) {
	detail, exposed = describeLoose(path)
	if !exposed {
		return "", "", false
	}
	return detail, fixCommand(path), true
}
