# Security Policy

For general information about security at Passbolt, see https://www.passbolt.com/security

## Reporting a security issue

If you've found a security issue in Passbolt, please do not open a public issue. Follow our responsible disclosure process: https://www.passbolt.com/docs/contribute/security/vulnerability/

## Where to store the config file

The config file holds your private key, its passphrase and your MFA seed in cleartext. Anyone who can read it can log in as you and decrypt every password you can see.

### What the CLI does

- It creates the config file and the KeePass export readable by you alone: mode `0600`, or on Windows an ACL granting only you, SYSTEM and Administrators[^windows-acl].
- It tightens a file whose permissions are too loose, then checks that the change took effect.
- It refuses to write to a file that stays loose, that is not yours, or that is not a regular file.

[^windows-acl]: SYSTEM is Windows itself, and an administrator can take ownership of any file whatever its ACL, so leaving them out would protect nothing. It is also the default ACL of your profile folder.

### What it does not do

File permissions only stop other users of **this machine**, and only on a disk that enforces them. The CLI does not check where the file is stored; that choice is yours.

Permissions do not protect the file in these places:

| Place | Who else can read it |
|---|---|
| USB sticks and external drives (FAT, exFAT) | anyone who plugs the drive in; these filesystems keep no permissions |
| Network shares (SMB, NFS) | the file server's administrators, and often other users of the share |
| VM or container shared folders | the host, and anything else with access to the shared folder |
| Cloud-synced folders (Dropbox, OneDrive, iCloud, ...) | the sync provider, and any device or person the folder is shared with |
| Backups and disk images | anyone who can read the backup |

### What to do

- Keep the file on a local disk, in your home directory. The default location is fine.
- If that is not possible, save only `serverAddress` and pass the rest through environment variables or flags. With no `userPassword` saved, the CLI asks for it each time.
