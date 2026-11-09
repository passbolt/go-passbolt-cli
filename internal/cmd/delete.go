package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/folder"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/group"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/resource"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/settings"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/user"
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
	deleteCmd.AddCommand(settings.NewDeleteCommand())

	deleteCmd.PersistentFlags().String("id", "", "ID of the Entity to Delete")
}
