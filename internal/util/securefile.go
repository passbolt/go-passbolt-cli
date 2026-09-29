package util

import (
	"fmt"
	"os"
)

// CreateSecretFile creates or truncates path as a regular file owned by the caller, mode 0600 or its Windows ACL equivalent.
func CreateSecretFile(path string) (file *os.File, err error) {
	// No truncation yet: a refusal below must leave the file as it was.
	file, err = openSecret(path)
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
	if !info.Mode().IsRegular() || !ownedByCaller(file) {
		return nil, fmt.Errorf("refusing to write %q: not a regular file you own", path)
	}
	// Creation alone secures only a new file, so an existing loose one is tightened here.
	if err = tightenIfLoose(file, info); err != nil {
		return nil, fmt.Errorf("securing %q: %w", path, err)
	}
	return file, file.Truncate(0)
}

// RepairMode tightens path to mode when anyone but the owner can reach it; a missing path
// is fine. Only the default config path and its directory reach this.
func RepairMode(path string, mode os.FileMode) error { return repairMode(path, mode) }

// WarnIfGroupOrWorldAccessible reports a config file others can read, without touching it.
func WarnIfGroupOrWorldAccessible(path string) {
	detail, loose := describeLoose(path)
	if !loose {
		return
	}
	Eprintf("Warning: %s is accessible to other users (%s).\n", Sanitize(path), Sanitize(detail))
	Eprintf("If it is a passbolt config it holds your private key and passphrase in cleartext; run: %s\n", Sanitize(fixCommand(path)))
}
