# go-passbolt-cli
A CLI tool to interact with Passbolt, an Open source Password Manager for teams.

If you want to do something more complicated: [this Go Module](https://github.com/passbolt/go-passbolt) to Interact with Passbolt from Go might interest you.


Disclaimer: This project is community-driven and not associated with Passbolt SA
# Install

## Via Repository (Prefered):
[![Packaging status](https://repology.org/badge/vertical-allrepos/go:passbolt-cli.svg)](https://repology.org/project/go:passbolt-cli/versions)

    Use the package from your Distros Official Repository

## Via Package:
    Download the deb/rpm Package for your Distro and architecture from the Latest Release.
    Install via your Distros Package manager like `dpkg -i`

## Via Homebrew
    brew install passbolt/tap/go-passbolt-cli

## Via Archive:
    Download and Extract the Archive for your OS and architecture from the Latest Release.
Note: tab completion and manpages will need to be installed manually.

## Via Go:
    go install github.com/passbolt/go-passbolt-cli@latest
Note: this will install the binary as go-passbolt-cli, also tab completion and manpages will be missing.

# Getting Started
First, you need to set up basic information: the Server Address, your Private Key, and your Password.
You have these options:
- Save it in the config file using
```
passbolt configure --serverAddress https://passbolt.example.org --userPassword '1234' --userPrivateKeyFile 'keys/privatekey.asc' 
```
or
```
passbolt configure --serverAddress https://passbolt.example.org --userPassword '1234' --userPrivateKey '-----BEGIN PGP PRIVATE KEY BLOCK-----' 
```
- Setup Environment Variables
- Provide the Flags manually every time

Notes:
- You can set the Private Key using the flags `--userPrivateKey` or `--userPrivateKeyFile` where `--userPrivateKey` takes the actual private key and `--userPrivateKeyFile` loads the content of a file as the PrivateKey, `--userPrivateKeyFile` overwrites the value of `--userPrivateKey`.
- You can also just store the serverAddress and your Private Key, if your Password is not set, it will prompt you for it every time.
- Passwordless PrivateKeys are unsupported
- MFA settings can also be saved permanently these ways

# Usage

Generally, the Structure of Commands is like this:
```bash
passbolt action entity [arguments]
```

Action is the Action you want to perform, like Creating, Updating, or Deleting an Entity.
Entity is a Resource(Password), Folder, User or Group that you want to apply an action to.

In Passbolt, a Password is usually referred to as a Resource.

To Create a Resource, you can do this. It will return the ID of the newly created Resource:
```bash
passbolt create resource --name "Test Resource" --password "Strong Password"
```

You can then list all users:
```bash
passbolt list user
```
Note: you can adjust which columns should be listed using the flag `--column` or its short from `-c`, if you want multiple columns then you need to specify this flag multiple times.


For sharing, we will need to know how we want to share. For that, there are these Permission Types:

| Code | Meaning | 
| --- | --- | 
| `1` | "Read-only" | 
| `7` | "Can update" | 
| `15` | "Owner" |
| `-1` | Delete existing permission | 

Now that we have a Resource ID, know the ID's of other Users and know about Permission Types, we can share the Resource with them:
```bash
passbolt share resource --id id_of_resource_to_share --type type_of_permission --user id_of_user_to_share_with
```
Note: you can supply the `user` argument multiple times to share with multiple users

For sharing with groups, the `--group` argument exists.

# MFA
You can also set up MFA using the configuration subcommand. Only TOTP is supported, and there are multiple modes for MFA: `none`, `interactive-totp`, and `noninteractive-totp`. 
| Mode | Description |
| --- | --- |
|`none`|just errors if challenged for MFA.
|`interactive-totp` | prompts for interactive entry of TOTP Codes.
|`noninteractive-totp` | automatically generates TOTP Codes when challenged. It requires the `mfaTotpToken` flag to be set to your totp Secret, you can configure the behavior using the `mfaDelay`, `mfaRetrys` and `mfaTotpOffset` flags


# Server Verification
To enable Server Verification, you need to run `passbolt verify` once. After that, the server will always be verified if the same config is used

# Scripting
For Scripting, we have a -j or --json flag to convert the Output for the create, get and list commands to JSON for easier Parsing in Scripts.

Note: The JSON Output does not cover Error Messages. You can detect Errors by checking if the Exitcode is not 0

# Documentation
Usage for all Subcommands is [here](https://github.com/passbolt/go-passbolt-cli/wiki/passbolt).
It is also available via `man passbolt`

