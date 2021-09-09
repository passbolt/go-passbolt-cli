# go-passbolt-cli
A CLI tool to interact with Passbolt, a Open source Password Manager for Teams.

If you want to do something more complicated [this](https://github.com/speatzle/go-passbolt) Go Module to Interact with Passbolt from Go Might Intrest you.

# Install

# Getting Started
Generally the Structure of Commands is like this:
```bash
go-passbolt-cli action entity [arguments]
```

Action is the Action you want to perform like Creating, Updating or Deleting a Entity.
Entity is a Resource(Password), Folder, User or Group that you want to apply a action to.

In Passbolt a Password is Usually Refert to as a Resource.

To Create a Resource you can do this, it will return the ID of the newly created Resource:
```bash
go-passbolt-cli create resource --name "Test Resource" --password "Strong Password"
```

You can then list all users:
```bash
go-passbolt-cli list user
```
For Sharing well need to know how we want to share, for that there are these Permission Types:

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

# Usage
Usage for all Subcommands is [here](https://github.com/speatzle/go-passbolt-cli/wiki/go-passbolt-cli).

