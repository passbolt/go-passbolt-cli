# go-passbolt-cli

A CLI tool to interact with [Passbolt](https://www.passbolt.com), an open source password manager for teams.

If you want to do something more complicated: [this Go module](https://github.com/passbolt/go-passbolt) to interact with Passbolt from Go might interest you.

# API Compatibility

The Go CLI is designed to work with the latest Passbolt API version, and we strongly recommend keeping your API up to date.

> [!IMPORTANT]
> Compatibility is currently tested with Passbolt API 5.6.1 and later. Older versions are unsupported and may not work.

Support for 5.6.1 is not guaranteed indefinitely and the minimum supported API version may increase in future releases.

In addition, users should also make sure they are running the latest version of the Go CLI. A current client stays compatible with supported older API versions, but an older client is not guaranteed to work against a newer API.

# Install

**Homebrew**

    brew install passbolt/tap/go-passbolt-cli

Homebrew installs the main `passbolt` CLI along with the helper binaries: `passbolt-browser-host`, `git-credential-passbolt`, and `docker-credential-passbolt`.

**Download Binary**

Download the deb/rpm package or archive for your OS and architecture from the [latest release](https://github.com/passbolt/go-passbolt-cli/releases/latest).

The packages include `passbolt`, `passbolt-browser-host`, `git-credential-passbolt`, and `docker-credential-passbolt`.

For deb/rpm, install via your distro's package manager (e.g. `dpkg -i`).

For archives, tab completion and manpages will need to be installed manually.

**Via Go**

```sh
go install github.com/passbolt/go-passbolt-cli@latest
go install github.com/passbolt/go-passbolt-cli/cmd/passbolt-browser-host@latest
```

Note: Tab completion and manpages will be missing when installing via `go install`.

## Community Packages

The following packages are maintained by the community and not by the Passbolt team.
See [Repology](https://repology.org/project/passbolt-cli/versions) for available packages and versions.

# Getting Started

First, you need to set up basic information:

- The server address,
- your private key
- and your password/passphrase.

You have these options:

- Save it in the config file using
 
```
passbolt configure --serverAddress https://passbolt.example.org --userPassword '1234' --userPrivateKeyFile 'keys/privatekey.asc' 
```

or

```
passbolt configure --serverAddress https://passbolt.example.org --userPassword '1234' --userPrivateKey '-----BEGIN PGP PRIVATE KEY BLOCK-----' 
```

- Set up environment variables
- Provide the flags manually every time

Notes:

- You can set the private key using the flags `--userPrivateKey` or `--userPrivateKeyFile` where `--userPrivateKey` takes the actual private key and `--userPrivateKeyFile` loads the content of a file as the private key, `--userPrivateKeyFile` overwrites the value of `--userPrivateKey`.
- You can also just store the `serverAddress` and your private key. If your password is not set it will prompt you for it every time.
- Passwordless private keys are not supported.
- MFA settings can also be saved permanently this way.

## The config file

`passbolt configure` writes to `$XDG_CONFIG_HOME/go-passbolt-cli/go-passbolt-cli.toml` (`~/.config/...` by default; `~/Library/Application Support/...` on macOS, `%AppData%\...` on Windows), or to the path given by `--config`.

**It stores what you give it in cleartext** — your private key, its passphrase, your MFA seed and the server verification tokens. Anyone who can read the file can log in as you and decrypt every password you can see, so it is written readable and writable by you alone (mode `0600`), in a `0700` directory. The CLI repairs those permissions on the default file if it finds them loosened, and warns about a `--config` file it cannot vouch for rather than changing it. Permissions cannot protect the file everywhere, such as on a USB stick, a network share or a synced folder: see [SECURITY.md](SECURITY.md) for where to keep it.

A symlink to one of your own files works, so dotfiles setups can keep the real file in a repository and link it into place; a link to a device or pipe is refused before anything is written, and a file another user owns is written with a warning on stderr.

If you would rather not keep the secrets on disk, store only `serverAddress` and supply the rest through environment variables or flags; with no `userPassword` saved the CLI prompts for it each time.

Note that `--debug` prints every stored setting, secrets included. It asks before doing so, and skips the dump entirely when it is not attached to a terminal unless you pass `--yes`.

## Interactive authentication agent (opt-in)

Use `passbolt --agent list user` to start an in-memory authentication agent. It
asks for approval for each command, and asks for your passphrase and TOTP when
needed. Later commands reuse the authenticated session. Authentication expires
after one hour without approved use; active use resets that idle timer. Server
session expiry and MFA policy still apply.

To enable this for your configuration, add these settings to your TOML file:

```toml
agent = true
agent-dialog = "auto"
```

Keep `userPassword` and your MFA seed out of the file to enter them interactively.
`--agent=false` uses ordinary CLI authentication for one invocation.

The default dialog backend uses Zenity or KDialog under X.org/Wayland, system
dialogs on macOS and Windows, or a full-terminal dialog when no desktop helper
is available. Install `zenity` or `kdialog` for Linux desktop dialogs. Select
`--agent-dialog=terminal` to always use the terminal, or
`--agent-dialog=pinentry --agent-pinentry=/path/to/pinentry` to use GnuPG's
pinentry helper. An explicit denial never falls back to another dialog.
Pipelines work: approval uses the desktop or controlling terminal, leaving the
command's stdin and stdout available for data. Without either UI, agent access
fails. `--yes` does not bypass approval.

Approvals identify the calling process and its parent where the operating system
provides that information. Executable names and PIDs are shown; command-line
arguments are never displayed because they may contain credentials. This is
best-effort context, not proof of requester identity. Reading a resource through
`get resource`, `exec` environment substitution, or a credential helper adds a resource-specific approval after its
metadata is loaded and before its secret is fetched. The dialog includes the
resource's full name in its title and body, its UUID, account, and command. If a
name is unavailable, the UUID remains visible. Bulk list/export operations retain
command-level approval.

```sh
passbolt --agent list user --json | jq .
passbolt agent status
passbolt agent lock   # clear the session matching this configuration
passbolt agent stop   # clear all cached sessions and stop the process
```

The agent runs Passbolt operations itself and keeps private keys and session
cookies inside its process. `exec` receives only the requested resource values
and launches your program in the foreground, preserving its terminal. Cached
sessions are isolated by server, account key and authentication/transport policy.
The agent never writes cached authentication to disk or a credential store.
`--debug` is unavailable in agent mode because SDK debug logging exposes secrets.
After upgrading the CLI, stop an older running agent before using the new binary.
See [SECURITY.md](SECURITY.md) for the local security boundary.

### Browser-based authorization (Browser Extension integration)

Instead of saving your private key, passphrase, or MFA seed in the local configuration file or passing them via environment variables, you can authorize CLI commands directly through your active session in the official Passbolt browser extension.

#### 1. Setup

Register the native messaging host manifest for your browser:

```sh
passbolt browser install
```

This installs the manifest for Chrome, Chromium, and Firefox in the current user's profile (`~/.config/...` on Linux, `~/Library/Application Support/...` on macOS, `%AppData%\...` on Windows).

To uninstall the manifest:

```sh
passbolt browser uninstall
```

#### 2. Usage

Run any command with `--agent --agent-auth=browser`:

```sh
passbolt list user --serverAddress https://passbolt.example.com --agent --agent-auth=browser
```

Or configure it in your `go-passbolt-cli.toml`:

```toml
serverAddress = "https://passbolt.example.com"
agent = true
agent-auth = "browser"
```

When a command needs authentication:
1. The CLI agent launches an ephemeral request and notifies your browser via native messaging.
2. The Passbolt extension automatically opens an "Authorize CLI Access" window displaying the server and requester context.
3. If your extension is locked, you can enter your passphrase directly to unlock it.
4. Click **Approve**. The authorization window closes automatically.
5. The extension encrypts your private key in memory using a single-use ECDH P-256 sealed envelope and sends it to the CLI agent.
6. The CLI agent decrypts the key in memory, logs in to the Passbolt server, and caches the authenticated session according to your agent idle timeout. The private key is never written to disk or the configuration file.

## Raw secret output for pipelines

Print one string field without tables, labels, JSON, or an added newline:

```sh
passbolt get resource --id RESOURCE_UUID --field password --raw
```

`--raw` defaults to the `password` field; `--field` selects another string field,
including a named custom field. Surrounding whitespace is trimmed. Authentication
uses the agent UI or a separate desktop/TTY dialog, leaving stdin available to the
calling process. Warnings and diagnostics go to stderr, with raw API error details
redacted. `--raw` cannot be combined with `--json` or `--debug`.

A consumer that implements `token_command` can invoke it directly. For example,
these **hcloud Part 2** configurations require token-command support in hcloud;
this Passbolt change does not implement or enable that hcloud setting:

```toml
[[contexts]]
name = "production"
token_command = "passbolt get resource --id PROD_RESOURCE_UUID --field password --raw"

[[contexts]]
name = "staging"
token_command = "passbolt get resource --id STAGING_RESOURCE_UUID --field password --raw"
```

Replace the placeholders with immutable UUIDs. API tokens remain in Passbolt;
they are not embedded in commands, environment variables, or consumer config.

## Native Git credentials

The release includes `git-credential-passbolt`. Configure Git with:

```sh
git config --global credential.helper passbolt
git config --global credential.useHttpPath true
```

The equivalent direct command is `passbolt credential-helper git get`.
`passbolt git-credential get` remains an alias for existing callers.

Git's `protocol`, `host`, optional `path` and username hint are matched against
Passbolt resource URI metadata. Match requires the same HTTP(S) scheme and exact
normalized host/port; a stored path must match on a slash boundary. If Git omits
the path, all same-origin credentials are considered. The helper reads metadata
first and decrypts only the selected resource's password. Multiple URI matches
from the same resource are deduplicated. Ambiguous matches require a controlling
terminal and explicit selection using the desktop, terminal, or pinentry UI;
unattended ambiguity cancels. `--yes` never bypasses selection.

Passbolt's server/account are configured separately. Passbolt permalinks and UUID
paths are **not resource selectors** in the Git protocol. No URI matching fallback
is used for the deterministic `get resource --id` command. A no-match Git lookup
returns no fields so Git can try another helper. `store` and `erase` are no-ops.

## Native Docker credentials

The release includes `docker-credential-passbolt`, using the official
`github.com/docker/docker-credential-helpers/credentials` interface and runtime.
Example Docker configuration:

```json
{
  "credHelpers": {
    "registry.example.com": "passbolt"
  }
}
```

`passbolt credential-helper docker get` accepts the registry server URL on stdin
and returns Docker's `ServerURL`, `Username` and `Secret` JSON. `list` returns the
standard server-to-username map and never decrypts passwords. Registry entries are
HTTP(S) origin URIs (also `/v2/` endpoints), with Docker Hub's standard aliases
normalized. Repository/web-page paths are excluded from Docker enumeration.
Conflicting usernames require interactive selection or cancel.

Docker's upstream client expects its fixed not-found status message on stdout
with a nonzero exit status when no credential matches. That standardized protocol
response is the only non-JSON Docker failure output; diagnostics stay on stderr.
`store` and `erase` return a read-only error and never change Passbolt resources.
Git and Docker preserve credential bytes within their wire formats; invalid Git
line-protocol characters are rejected before output. `--debug` is unavailable.

## AWS credential_process

```ini
[profile passbolt]
credential_process = passbolt credential-helper aws get --id RESOURCE_UUID
```

The AWS version-1 response maps the resource username to `AccessKeyId` and password
to `SecretAccessKey`. Optional string fields `session_token` and `expiration` are
read from encrypted resource/custom fields; expiration is an RFC3339 value.
Missing optional values are omitted, not serialized as null. Customize field names
with `--session-token-field` and `--expiration-field`; an empty name disables the
optional field. An explicitly configured field must exist. Invalid or expired
credentials fail before any JSON is emitted. Custom fields may be selected by
name or immutable field ID; keep session tokens in secret values, not metadata.

## Kubernetes exec authentication

Example kubeconfig user:

```yaml
users:
  - name: passbolt
    user:
      exec:
        apiVersion: client.authentication.k8s.io/v1
        command: passbolt
        args: [credential-helper, k8s, get, --id, RESOURCE_UUID]
        interactiveMode: IfAvailable
```

The helper emits the official `client-go` v1 `ExecCredential` type with the
resource password in `status.token` and an optional RFC3339 expiration field.
`KUBERNETES_EXEC_INFO` is validated; a host declaring `interactive: false` cannot
trigger agent approval or authentication prompts. Agent use requires interaction,
so that combination fails clearly. All cloud helpers use immutable resource UUIDs,
never service-URL discovery, and keep lookup/cleanup errors free of secret data.

# Usage

Generally, the structure of commands are like this:

```bash
passbolt action entity [arguments]
```

`action` is the action you want to perform like creating, updating or deleting an entity.
`entity` is a resource (e.g. password), folder, user or group that you want to apply an action to.

In Passbolt a password is usually referred to as a "resource".

To create a resource you can do the following, which will return the ID of the newly created resource:

```bash
passbolt create resource --name "Test Resource" --password "Strong Password"
```

You can then list all users:

```bash
passbolt list user
```

Note: You can adjust which columns should be listed using the flag `--column` or its short from `-c`,
if you want multiple column then you need to specify this flag multiple times.

For sharing, we will need to know how we want to share, for that there are these permission types:

| Code | Meaning                    | 
|------|----------------------------| 
| `1`  | Read-only                  | 
| `7`  | Can update                 | 
| `15` | Owner                      |
| `-1` | Delete existing permission | 

Now, that we have a resource ID, know the IDs of other users and know about permission types, we can share the resource with them:

```bash
passbolt share resource --id id_of_resource_to_share --type type_of_permission --user id_of_user_to_share_with
```

Note: You can supply the users argument multiple times to share with multiple users.

For sharing with groups the `--group` argument exists.

# MFA

You can set up MFA also using the configuration sub command. Only TOTP is supported. There are multiple modes for MFA: `none`, `interactive-totp` and `noninteractive-totp`.

| Mode                  | Description                                                                                                                                                                                                       |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `none`                | just errors if challenged for MFA.                                                                                                                                                                                |
| `interactive-totp`    | prompts for interactive entry of TOTP Codes.                                                                                                                                                                      |
| `noninteractive-totp` | automatically generates TOTP codes when challenged. It requires the `mfaTotpToken` flag to be set to your TOTP secret. You can configure the behavior using the `mfaDelay`, `mfaRetrys` and `mfaTotpOffset` flags |

# Server Verification

To enable server verification, you need to run `passbolt verify` once, after that the server will always be verified if the same config is used.

# Scripting

For scripting we have a `-j` or `--json` flag to convert the output for the `create`, `get` and `list` commands to JSON for easier parsing in scripts.

Note: The JSON output does not cover error messages. You can detect errors by checking if the exit code is not 0.

# Exposing Secrets to Subprocesses

The `exec` command allows you to execute another command with environment variables that reference secrets stored in Passbolt.
Any environment variables containing `passbolt://` references are automatically resolved to their corresponding secret values
before the specified command is executed. This ensures that secrets are securely injected into the child process's environment
without exposing them to the parent shell.

For example:

```bash
export GITHUB_TOKEN=passbolt://<PASSBOLT_RESOURCE_ID_HERE>
passbolt exec -- gh auth login
```

This would resolve the `passbolt://` reference in `GITHUB_TOKEN` to its actual secret value and pass it to the GitHub process.

# Documentation

Usage for all subcommands is [here](https://github.com/passbolt/go-passbolt-cli/wiki/passbolt).
And is also available via `man passbolt`
