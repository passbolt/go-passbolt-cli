//go:build !windows

package util

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// secretMode is the mode for files holding credentials in cleartext.
const secretMode os.FileMode = 0o600

// hasPOSIXModes reports whether permission bits mean anything here.
const hasPOSIXModes = true

// isLoose reports whether anyone but the owner can reach the file.
func isLoose(info fs.FileInfo) bool { return info.Mode().Perm()&0o077 != 0 }

// openSecret opens path for writing, creating it 0600; O_NONBLOCK so a named pipe cannot hang the open.
func openSecret(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|syscall.O_NONBLOCK, secretMode)
}

// ownedByCaller reports whether the open file belongs to the user running this process.
func ownedByCaller(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Getuid()
}

// tightenIfLoose drops group and world access; O_CREATE sets the mode only on creation.
func tightenIfLoose(file *os.File, info fs.FileInfo) error {
	if !isLoose(info) {
		return nil
	}
	return file.Chmod(secretMode)
}

func repairMode(path string, mode os.FileMode) error {
	info, err := os.Stat(path) // follows a dotfiles symlink to its target
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil || !isLoose(info) {
		return err
	}
	return os.Chmod(path, mode) // EPERM if the target is not ours, which the caller warns about
}

// describeLoose reports whether others can reach path, and how.
func describeLoose(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || !isLoose(info) {
		return "", false
	}
	return fmt.Sprintf("mode %04o", info.Mode().Perm()), true
}

func fixCommand(path string) string { return "chmod 600 " + path }
