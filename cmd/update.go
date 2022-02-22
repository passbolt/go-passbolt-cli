package cmd

import (
	"github.com/passbolt/go-passbolt-cli/folder"
	"github.com/passbolt/go-passbolt-cli/group"
	"github.com/passbolt/go-passbolt-cli/resource"
	"github.com/passbolt/go-passbolt-cli/user"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Updates a Passbolt Entity",
	Long:    `Updates a Passbolt Entity`,
	Aliases: []string{"change"},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(resource.ResourceUpdateCmd)
	updateCmd.AddCommand(folder.FolderUpdateCmd)
	updateCmd.AddCommand(group.GroupUpdateCmd)
	updateCmd.AddCommand(user.UserUpdateCmd)
}
