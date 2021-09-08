package cmd

import (
	"github.com/speatzle/go-passbolt-cli/folder"
	"github.com/speatzle/go-passbolt-cli/group"
	"github.com/speatzle/go-passbolt-cli/resource"
	"github.com/speatzle/go-passbolt-cli/user"
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
	getCmd.AddCommand(resource.ResourceGetCmd)
	getCmd.AddCommand(folder.FolderGetCmd)
	getCmd.AddCommand(group.GroupGetCmd)
	getCmd.AddCommand(user.UserGetCmd)

}
