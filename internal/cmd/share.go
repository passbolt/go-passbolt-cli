package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/folder"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/resource"
	"github.com/spf13/cobra"
)

// shareCmd represents the share command
var shareCmd = makeshareCmd()

func makeshareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "share",
		Short: "Shares a Passbolt Entity",
		Long:  `Shares a Passbolt Entity`,
	}
}

func configureShare() {
	rootCmd.AddCommand(shareCmd)
	shareCmd.AddCommand(resource.ResourceShareCmd)
	shareCmd.AddCommand(folder.FolderShareCmd)
}

func init() { configureShare() }
