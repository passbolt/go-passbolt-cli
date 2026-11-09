package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/folder"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/group"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/resource"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/user"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = makecreateCmd()

func makecreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "create",
		Short:   "Creates a Passbolt Entity",
		Long:    `Creates a Passbolt Entity`,
		Aliases: []string{"new"},
	}
}

func configureCreate() {
	rootCmd.AddCommand(createCmd)
	createCmd.PersistentFlags().BoolP("json", "j", false, "Output JSON")
	createCmd.AddCommand(resource.ResourceCreateCmd)
	createCmd.AddCommand(folder.FolderCreateCmd)
	createCmd.AddCommand(group.GroupCreateCmd)
	createCmd.AddCommand(user.UserCreateCmd)
}

func init() { configureCreate() }
