package cmd

import (
	"github.com/passbolt/go-passbolt-cli/folder"
	"github.com/passbolt/go-passbolt-cli/group"
	"github.com/passbolt/go-passbolt-cli/resource"
	"github.com/passbolt/go-passbolt-cli/user"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:     "get",
	Short:   "Gets a Passbolt Entity",
	Long:    `Gets a Passbolt Entity`,
	Aliases: []string{"read"},
}

func init() {
	rootCmd.AddCommand(getCmd)
    getCmd.PersistentFlags().BoolP("json", "j", false, "Outputs JSON")
	getCmd.AddCommand(resource.ResourceGetCmd)
	getCmd.AddCommand(folder.FolderGetCmd)
	getCmd.AddCommand(group.GroupGetCmd)
	getCmd.AddCommand(user.UserGetCmd)

}
