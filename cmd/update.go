package cmd

import (
	"github.com/speatzle/go-passbolt-cli/folder"
	"github.com/speatzle/go-passbolt-cli/group"
	"github.com/speatzle/go-passbolt-cli/resource"
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
}
