package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/folder"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/group"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/resource"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/settings"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/user"
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
	getCmd.PersistentFlags().BoolP("json", "j", false, "Output JSON")
	getCmd.AddCommand(resource.ResourceGetCmd)
	getCmd.AddCommand(folder.FolderGetCmd)
	getCmd.AddCommand(group.GroupGetCmd)
	getCmd.AddCommand(user.UserGetCmd)
	getCmd.AddCommand(settings.NewGetCommand())

}
