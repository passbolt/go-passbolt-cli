//go:build !windows

package securefile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"github.com/passbolt/go-passbolt-cli/internal/util/securefile/posixmode"
)

// hasPOSIXModes reports whether permission bits mean anything here.
const hasPOSIXModes = true

// openSecret opens path for writing, creating it 0600; O_NONBLOCK so a named pipe cannot hang the open.
func openSecret(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|syscall.O_NONBLOCK, posixmode.SecretMode)
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

// fchmod and chmodPath are the two chmods, swappable so a test can play a filesystem that ignores them.
// They are test seams, not part of the platform contract.
var (
	fchmod    = (*os.File).Chmod
	chmodPath = os.Chmod
)

// tightenIfLoose drops group and world access; O_CREATE sets the mode only on creation.
func tightenIfLoose(file *os.File, info fs.FileInfo) error {
	if !posixmode.IsLoose(info) {
		return nil
	}
	if err := fchmod(file, posixmode.SecretMode); err != nil {
		return err
	}
	// Read back: some filesystems accept chmod and drop it.
	after, err := file.Stat()
	if err != nil {
		return err
	}
	if posixmode.IsLoose(after) {
		return errPermissionsIgnored
	}
	return nil
}

func repairMode(path string, mode os.FileMode) error {
	info, err := os.Stat(path) // follows a dotfiles symlink to its target
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil || !posixmode.IsLoose(info) {
		return err
	}
	if err := chmodPath(path, mode); err != nil {
		return err // EPERM if the target is not ours, which the caller warns about
	}
	// By path, like the chmod: opening it could block on a FIFO or fail on a write-only mode.
	if after, err := os.Stat(path); err != nil {
		return err
	} else if posixmode.IsLoose(after) {
		return errPermissionsIgnored
	}
	return nil
}

// describeLoose reports whether others can reach path, and how.
func describeLoose(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || !posixmode.IsLoose(info) {
		return "", false
	}
	return fmt.Sprintf("mode %04o", info.Mode().Perm()), true
}

func fixCommand(path string) string { return "chmod 600 " + path }
