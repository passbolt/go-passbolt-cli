package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/folder"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/group"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/resource"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/user"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = makeupdateCmd()

func makeupdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "update",
		Short:   "Updates a Passbolt Entity",
		Long:    `Updates a Passbolt Entity`,
		Aliases: []string{"change"},
	}
}

func configureUpdate() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(resource.ResourceUpdateCmd)
	updateCmd.AddCommand(folder.FolderUpdateCmd)
	updateCmd.AddCommand(group.GroupUpdateCmd)
	updateCmd.AddCommand(user.UserUpdateCmd)
}

func init() { configureUpdate() }
