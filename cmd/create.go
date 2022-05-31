package cmd

import (
	"github.com/passbolt/go-passbolt-cli/folder"
	"github.com/passbolt/go-passbolt-cli/group"
	"github.com/passbolt/go-passbolt-cli/resource"
	"github.com/passbolt/go-passbolt-cli/user"
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
	createCmd.PersistentFlags().BoolP("json", "j", false, "Outputs JSON")
	createCmd.AddCommand(resource.ResourceCreateCmd)
	createCmd.AddCommand(folder.FolderCreateCmd)
	createCmd.AddCommand(group.GroupCreateCmd)
	createCmd.AddCommand(user.UserCreateCmd)
}
