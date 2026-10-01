// Package securefile creates and checks files that hold secrets: the CLI config, which
// holds the private key and passphrase in cleartext, and the KeePass export.
//
// Create writes them at 0600 and RepairMode fixes the default config (a --config path is
// only reported, through Exposure). On Windows a protected ACL granting only the caller,
// SYSTEM and Administrators replaces 0600. A mode or ACL still loose after tightening is
// refused rather than written. Where the file is stored is the user's choice; SECURITY.md
// says which places permissions cannot protect.
//
// securefile_unix.go and securefile_windows.go implement the same platform functions, listed
// in securefile.go; their helpers live in posixmode and winacl.
//
// The package writes nothing to the terminal, so it can sit below util.
package securefile
