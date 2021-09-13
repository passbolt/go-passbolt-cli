# go-passbolt-cli
A CLI tool to interact with Passbolt, a Open source Password Manager for Teams.

If you want to do something more complicated: [this](https://github.com/speatzle/go-passbolt) Go Module to Interact with Passbolt from Go might intrest you.

# Install

## Via Package (Prefered):
    Download the Package for your OS and architecture from the Latest Release.
    Install via your Distros Package manager like `dpkg -i`

## Via Archive:
    Download and Extract the Archive for your OS and architecture from the Latest Release.
Note: tab completion and manpages will need to be installed manually.

## Via Go:
    go install github.com/speatzle/go-passbolt-cli
Note: this will install the binary as go-passbolt-cli, also tab completion and manpages will be missing.

# Getting Started
First you need to Setup basic information: the Server Address, your Private Key and your Password.
You have these options:
- Save it in the config file using `passbolt configure --serverAddress https://passbolt.example.org --userPrivateKey 'private' --userPassword '1234'`
- Setup Enviroment Variables
- Provide the Flags manually every time

Note: you can also just store the serverAddress and your Private Key, if your Password is not set it will prompt you for it every time

# Usage

Generally the Structure of Commands is like this:
```bash
go-passbolt-cli action entity [arguments]
```

Action is the Action you want to perform like Creating, Updating or Deleting a Entity.
Entity is a Resource(Password), Folder, User or Group that you want to apply a action to.

In Passbolt a Password is usually revert to as a Resource.

To Create a Resource you can do this, it will return the ID of the newly created Resource:
```bash
go-passbolt-cli create resource --name "Test Resource" --password "Strong Password"
```

You can then list all users:
```bash
go-passbolt-cli list user
```
For sharing we will need to know how we want to share, for that there are these Permission Types:

| Code | Meaning | 
| --- | --- | 
| `1` | "Read-only" | 
| `7` | "Can update" | 
| `15` | "Owner" |
| `-1` | Delete existing permission | 

Now that we have a Resource ID, know the ID's of other Users and about know about Permission Types, we can share the Resource with them:
```bash
go-passbolt-cli share resource --id id_of_resource_to_share --type type_of_permission --users id_of_user_to_share_with
```
Note: you can supply the the users argument multiple times to share with multiple users

For sharing with groups the `--groups` argument exists.

# Documentation
Usage for all Subcommands is [here](https://github.com/speatzle/go-passbolt-cli/wiki/go-passbolt-cli).
And is also available via `man passbolt`

