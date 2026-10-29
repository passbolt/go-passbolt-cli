# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

Releases before this file was introduced are documented in the [GitHub releases](https://github.com/passbolt/go-passbolt-cli/releases).

## [Unreleased]

## [0.5.3] - 2026-10-29
### Added
- PB-54610: Skip Resources this build cannot read in `list resource`
- PB-54610: Explain schema validation errors in `create`, `update`, `share` and `get`

### Fixed
- PB-54147: Propagate the real error when interactive TOTP is exhausted

### Security
- PB-54765: Sanitize terminal output in every command to prevent escape sequence injection (Medium)
- PB-54765: Escape non-BMP characters as `\uXXXX` in JSON output
- PB-54766: Write the CLI config through a hardened file descriptor (Medium)
- PB-54766: Repair default config permissions and warn on custom paths
- PB-54766: Create the KeePass export with owner-only permissions
- PB-54766: Ask for confirmation before `configure --debug` prints the saved settings, and print them to stderr
- PB-54766: Document cleartext credential storage in the config file
- PB-54978: Restrict the config file and the KeePass export to their owner on Windows
- PB-54978: Document where the config file is unsafe to store
- PB-54987: Warn instead of refusing when a secret file stays readable by others or belongs to someone else
- PB-54147: Move to the Go 1.27.1 security patch
- PB-55389: Update Go to 1.27.2 (security fixes in crypto/tls, net/http, net/textproto and os)
- PB-55610: Bump Go version to 1.27.3 (security fixes in crypto/x509)

### Maintenance
- PB-54610: Bump the Go SDK to v0.8.4
- PB-54147: Bump Go to 1.27 (the Go SDK stays buildable on Go 1.26.10)
- PB-54147: Pin the omitempty semantics the `--json` output relies on
- PB-54765: Forbid direct fmt printing in golangci-lint
- PB-54765: Add integration tests to avoid terminal escape regression
- PB-54766: Add regression tests for config file permissions
- PB-54978: Move the secure-file code into util/securefile
- PB-54610: Run the txtar scenarios as benchmarks
- PB-54610: Cover every bundled resource type end to end
- PB-54610: Reorganize integration test runner
- PB-55287: Reorganize e2e test files in CLI and SDK
- PB-54998: Update GitHub Actions to stable versions
- PB-55888: Update Go dependencies
- PB-55888: Migrate cel-go to the cel.dev/cel-go module path
- PB-55888: Keep the previous table style after the pterm upgrade
- Renovate: Update goreleaser/goreleaser-action action to v7.3.0
- Renovate: Update actions/upload-artifact action to v7.0.2

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
