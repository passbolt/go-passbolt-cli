package cmd

import (
	"github.com/passbolt/go-passbolt-cli/folder"
	"github.com/passbolt/go-passbolt-cli/group"
	"github.com/passbolt/go-passbolt-cli/resource"
	"github.com/passbolt/go-passbolt-cli/user"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Deletes a Passbolt Entity",
	Long:    `Deletes a Passbolt Entity`,
	Aliases: []string{"remove"},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(resource.ResourceDeleteCmd)
	deleteCmd.AddCommand(folder.FolderDeleteCmd)
	deleteCmd.AddCommand(group.GroupDeleteCmd)
	deleteCmd.AddCommand(user.UserDeleteCmd)

	deleteCmd.PersistentFlags().String("id", "", "ID of the Entity to Delete")
}
