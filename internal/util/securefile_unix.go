//go:build !windows

package util

import (
	"io/fs"
	"os"
	"syscall"
)

// oNonBlock stops a named pipe blocking the open.
const oNonBlock = syscall.O_NONBLOCK

// hasPOSIXModes reports whether permission bits mean anything here.
const hasPOSIXModes = true

// ownedByCaller reports whether info belongs to the user running this process.
func ownedByCaller(info fs.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Getuid()
}
