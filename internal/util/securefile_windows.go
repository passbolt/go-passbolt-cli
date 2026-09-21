//go:build windows

package util

import "io/fs"

// Windows has no O_NONBLOCK and secures files by ACL, so both of these are inert here.
const oNonBlock = 0
const hasPOSIXModes = false

// ownedByCaller cannot be read off the mode bits here, where an ACL decides instead.
func ownedByCaller(fs.FileInfo) bool { return true }
