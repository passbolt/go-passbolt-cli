go-passbolt-cli v0.5.3 is a security-focused release. It protects the terminal against escape sequence injection, hardens how the configuration file and the KeePass export are written on every platform, and brings schema-aware Resource commands from the Go SDK v0.8.4.

## Terminal output sanitizing
Resource names, usernames, URIs and other server-provided values were printed to the terminal as they were. A value containing escape sequences could rewrite what the user saw or send commands to the terminal emulator. Every command now sanitizes what it prints, and the `--json` output escapes non-BMP characters as `\uXXXX`.

## Configuration file and export hardening
The configuration file holds the private key and its passphrase in cleartext. It is now created and kept with owner-only permissions: 0600 on Unix, and on Windows a protected ACL granting only the caller, SYSTEM and Administrators. The KeePass export gets the same treatment. The CLI refuses to write either one to anything but a regular file, and warns when the file stays readable by others or belongs to someone else. The default configuration file is repaired when its permissions are loose. SECURITY.md documents where the configuration file is unsafe to store.

`configure --debug` now asks for confirmation before it prints the saved settings, secrets included, and prints them to stderr so they stay out of a piped output.

## Schema-aware Resource commands
With the Go SDK v0.8.4, Resources are validated against schemas bundled with the SDK. `list resource` skips Resources whose type is unknown to this build or disabled on the server and reports how many were ignored, while `create`, `update`, `share` and `get` explain why a Resource failed validation.

## Go version and dependencies
The CLI is built with Go 1.27.3, which includes the Go security fixes in crypto/tls, net/http, net/textproto, os and crypto/x509. Dependencies are updated, and cel-go moves to its new `cel.dev/cel-go` module path. Table output keeps its previous layout after the pterm upgrade.

## Conclusion
Many thanks to everyone who provided feedback, reported bugs, and contributed to making Passbolt better!

## Changelogs
### Added
- PB-54610 Skip Resources this build cannot read in `list resource`
- PB-54610 Explain schema validation errors in `create`, `update`, `share` and `get`

### Fixed
- PB-54147 Propagate the real error when interactive TOTP is exhausted

### Security
- PB-54765 Sanitize terminal output in every command to prevent escape sequence injection
- PB-54765 Escape non-BMP characters as `\uXXXX` in JSON output
- PB-54766 Write the CLI config through a hardened file descriptor (Medium)
- PB-54766 Repair default config permissions and warn on custom paths
- PB-54766 Create the KeePass export with owner-only permissions
- PB-54766 Ask for confirmation before `configure --debug` prints the saved settings, and print them to stderr
- PB-54766 Document cleartext credential storage in the config file
- PB-54978 Restrict the config file and the KeePass export to their owner on Windows
- PB-54978 Document where the config file is unsafe to store
- PB-54987 Warn instead of refusing when a secret file stays readable by others or belongs to someone else
- PB-54147 Move to the Go 1.27.1 security patch
- PB-55389 Update Go to 1.27.2 (security fixes in crypto/tls, net/http, net/textproto and os)
- PB-55610 Bump Go version to 1.27.3 (security fixes in crypto/x509)

### Maintenance
- PB-54610 Bump the Go SDK to v0.8.4
- PB-54147 Bump Go to 1.27 (the Go SDK stays buildable on Go 1.26.10)
- PB-54147 Pin the omitempty semantics the `--json` output relies on
- PB-54765 Forbid direct fmt printing in golangci-lint
- PB-54765 Add integration tests to avoid terminal escape regression
- PB-54766 Add regression tests for config file permissions
- PB-54978 Move the secure-file code into util/securefile
- PB-54610 Run the txtar scenarios as benchmarks
- PB-54610 Cover every bundled resource type end to end
- PB-54610 Reorganize integration test runner
- PB-55287 Reorganize e2e test files in CLI and SDK
- PB-54998 Update GitHub Actions to stable versions
- PB-55888 Update Go dependencies
- PB-55888 Migrate cel-go to the cel.dev/cel-go module path
- PB-55888 Keep the previous table style after the pterm upgrade
- Renovate: Update goreleaser/goreleaser-action action to v7.3.0
- Renovate: Update actions/upload-artifact action to v7.0.2
