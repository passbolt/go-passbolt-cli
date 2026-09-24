package util

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// secretMode is the mode for files holding credentials in cleartext.
const secretMode os.FileMode = 0o600

// isLoose reports whether anyone but the owner can reach the file.
func isLoose(info fs.FileInfo) bool { return info.Mode().Perm()&0o077 != 0 }

// CreateSecretFile creates or truncates path as a regular file owned by the caller, mode 0600.
func CreateSecretFile(path string) (file *os.File, err error) {
	// O_NONBLOCK so a named pipe cannot hang the open. No O_TRUNC: truncation waits until
	// the descriptor is vouched for, so a refusal leaves the file as it was.
	file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|oNonBlock, secretMode)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			file.Close()
			file = nil
		}
	}()

	// Checked on the open descriptor, not on the path: nothing can be swapped in between.
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || !ownedByCaller(info) {
		return nil, fmt.Errorf("refusing to write %q: not a regular file you own", path)
	}
	// O_CREATE sets the mode only on creation, so an existing 0644 file would keep it.
	if hasPOSIXModes && isLoose(info) {
		if err = file.Chmod(secretMode); err != nil {
			return nil, fmt.Errorf("securing %q: %w", path, err)
		}
	}
	return file, file.Truncate(0)
}

// RepairMode tightens path to mode when anyone but the owner can reach it; a missing path
// is fine. Only the default config path and its directory reach this.
func RepairMode(path string, mode os.FileMode) error {
	if !hasPOSIXModes {
		return nil
	}
	info, err := os.Stat(path) // follows a dotfiles symlink to its target
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil || !isLoose(info) {
		return err
	}
	return os.Chmod(path, mode) // EPERM if the target is not ours, which the caller warns about
}

// WarnIfGroupOrWorldAccessible reports a config file others can read, without touching it.
func WarnIfGroupOrWorldAccessible(path string) {
	info, err := os.Stat(path)
	if !hasPOSIXModes || err != nil || !isLoose(info) {
		return
	}
	Eprintf("Warning: %s is accessible to other users (mode %04o).\n", Sanitize(path), info.Mode().Perm())
	Eprintf("If it is a passbolt config it holds your private key and passphrase in cleartext; run: chmod 600 %s\n", Sanitize(path))
}
