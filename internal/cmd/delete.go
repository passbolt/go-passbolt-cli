package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/folder"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/group"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/resource"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/user"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = makedeleteCmd()

func makedeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete",
		Short:   "Deletes a Passbolt Entity",
		Long:    `Deletes a Passbolt Entity`,
		Aliases: []string{"remove"},
	}
}

func configureDelete() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(resource.ResourceDeleteCmd)
	deleteCmd.AddCommand(folder.FolderDeleteCmd)
	deleteCmd.AddCommand(group.GroupDeleteCmd)
	deleteCmd.AddCommand(user.UserDeleteCmd)

	deleteCmd.PersistentFlags().String("id", "", "ID of the Entity to Delete")
}

func init() { configureDelete() }
