# go-passbolt-cli

A CLI tool to interact with [Passbolt](https://www.passbolt.com), an open source password manager for teams.

If you want to do something more complicated: [this Go module](https://github.com/passbolt/go-passbolt) to interact with Passbolt from Go might interest you.

Disclaimer: This project is community driven and not associated with [Passbolt SA](https://www.passbolt.com/about).

# Install

## Via Repository (Preferred):

[![Packaging status](https://repology.org/badge/vertical-allrepos/go:passbolt-cli.svg)](https://repology.org/project/go:passbolt-cli/versions)

Use the package from your distros official repository.

## Via Package:

Download the deb/rpm package for your distro and architecture from the latest release.

Install via your distros package manager like `dpkg -i`.

## Via Homebrew

    brew install passbolt/tap/go-passbolt-cli

## Via Archive:

Download and extract the archive for your OS and architecture from the latest release.

Note: Tab completion and manpages will need to be installed manually.

## Via Go:

    go install github.com/passbolt/go-passbolt-cli@latest

Note: This will install the binary as `go-passbolt-cli`. Also, tab completion and manpages will be missing.

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
