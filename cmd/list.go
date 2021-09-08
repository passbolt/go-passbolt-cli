package cmd

import (
	"github.com/speatzle/go-passbolt-cli/folder"
	"github.com/speatzle/go-passbolt-cli/group"
	"github.com/speatzle/go-passbolt-cli/resource"
	"github.com/speatzle/go-passbolt-cli/user"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "Lists Passbolt Entitys",
	Long:    `Lists Passbolt Entitys`,
	Aliases: []string{"index", "ls", "filter", "search"},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(resource.ResourceListCmd)
	listCmd.AddCommand(folder.FolderListCmd)
	listCmd.AddCommand(group.GroupListCmd)
	listCmd.AddCommand(user.UserListCmd)
}
