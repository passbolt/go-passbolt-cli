package cmd

import (
	"github.com/speatzle/go-passbolt-cli/folder"
	"github.com/speatzle/go-passbolt-cli/group"
	"github.com/speatzle/go-passbolt-cli/resource"
	"github.com/speatzle/go-passbolt-cli/user"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:     "create",
	Short:   "Creates a Passbolt Entity",
	Long:    `Creates a Passbolt Entity`,
	Aliases: []string{"new"},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(resource.ResourceCreateCmd)
	createCmd.AddCommand(folder.FolderCreateCmd)
	createCmd.AddCommand(group.GroupCreateCmd)
	createCmd.AddCommand(user.UserCreateCmd)
}
