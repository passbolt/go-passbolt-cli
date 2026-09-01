# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

Releases before this file was introduced are documented in the [GitHub releases](https://github.com/passbolt/go-passbolt-cli/releases).

## [Unreleased]
### Added
- KeePass export mirrors the Passbolt folder hierarchy as nested groups; folderless resources stay in the top group (#57, #138)
- KeePass export: `--folder` exports only the given folder and its subfolders (#75)
- KeePass export carries Passbolt tags, creation/modification timestamps and expiry dates into the KDBX entries
- KeePass export writes TOTP configuration in both conventions: the otpauth `otp` field (KeePassXC, KeeOtp2) and the native `TimeOtp-*` fields (KeePass 2.x built-in generator)
- KeePass export: `--comments` includes the Passbolt comment threads in the KeePass Notes field
- Interactive password prompts mask input with '*'; the KeePass export password is asked twice for confirmation

### Fixed
- KeePass export: harden the KDF settings (600k AES-KDF rounds for v3, 64 MiB Argon2 memory for v4) so KeePass no longer warns about weak key transformation
- KeePass export: an empty password is rejected instead of silently writing an unprotected kdbx
- KeePass export: a resource without a secret is skipped instead of crashing the export
- KeePass export: a failed export no longer leaves a partial .kdbx behind (write to temp file, rename on success), and write errors on close are reported

### Changed
- KeePass export: the top-level group is named "Passbolt Export" instead of "root"

## [0.5.2] - 2026-08-28
### Fixed
- PB-53937: Sign shared v5 metadata with both the user key and the metadata key (Go SDK v0.8.3)

### Security
- PB-53335: Fix crypto/tls CVE-2026-42505 (Medium)
- PB-53334: Minor upgrade for golang.org/x/text (High)
- PB-53929: Minor upgrade for github.com/moby/go-archive (High)
- PB-53356: Minor upgrade for github.com/google/cel-go (Medium)

### Maintenance
- PB-53915: Use the SDK custom field helpers in the KeePass export (Go SDK v0.8.2)
- PB-53919: Bump go version to 1.26.6
- PB-54017: Update dependencies in SDK and CLI
- PB-54017: Replace a deprecated gokeepasslib call in the KeePass export
- PB-54022: Use go 1.25 syntax
- PB-53686: Document the minimum supported Passbolt API version
- PB-54031: Add changelog files to SDK and CLI
- PB-53914: Remove the community disclaimer from the README
- PB-53788: Update devcontainer version in go CI
- PB-53937: Sign the metadata private key payload with a timestamp when bootstrapping the test environment
- Renovate: Add renovate.json
- Renovate: Adapt renovate config
