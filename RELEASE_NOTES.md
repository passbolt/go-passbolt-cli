go-passbolt-cli v0.5.3 fixes two medium-severity vulnerabilities, a terminal escape sequence injection and insecure configuration file permissions, and brings schema-aware Resource commands from the Go SDK v0.8.4.

## Security improvements
Security researcher Nitin Gavhane reported the following two vulnerabilities, both affecting versions before 0.5.3.

Some output paths printed server-controlled values, such as metadata field names and group names, without filtering terminal control characters, so a crafted Resource or group could forge the CLI output (CVSS 5.1). All terminal output is now sanitized.

The configuration file, which holds the private key, passphrase and MFA seed, could be readable by other local users, notably with a custom `--config` path (CVSS 6.8). It is now kept owner-only (0600, or an owner-only ACL on Windows), as is the KeePass export, and `configure --debug` asks before printing secrets.

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
- PB-54765 Sanitize terminal output in every command to prevent escape sequence injection (Medium)
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
