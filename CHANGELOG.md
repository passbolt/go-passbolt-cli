# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

Releases before this file was introduced are documented in the [GitHub releases](https://github.com/passbolt/go-passbolt-cli/releases).

## [Unreleased]
### Maintenance
- PB-54147: Bump go version to 1.27.0 (the minimum supported Go version is now 1.27)
- PB-54147: Use go 1.27 syntax (the go-passbolt SDK :stays buildable on Go 1.26.7, so this only affects the CLI)

### Changed
- PB-54610: `list resource` skips Resources whose type is unknown to this build or disabled on the server and reports how many were ignored, and `create`, `update`, `share` and `get` explain schema validation failures (Go SDK PB-54610)

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
