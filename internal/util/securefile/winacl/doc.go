// Package winacl holds the Windows ACL helpers behind securefile: the caller's SIDs, the
// private descriptor that stands in for 0600, and the check that a DACL exposes nobody else,
// which reads the DACL through the SDDL parser in sddl.go.
package winacl
