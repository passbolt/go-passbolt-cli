package cmd

import (
	"github.com/speatzle/go-passbolt-cli/resource"
	"github.com/spf13/cobra"
)

// moveCmd represents the move command
var moveCmd = &cobra.Command{
	Use:   "move",
	Short: "Moves a Passbolt Entity",
	Long:  `Moves a Passbolt Entity`,
}

func init() {
	rootCmd.AddCommand(moveCmd)
	moveCmd.AddCommand(resource.ResourceMoveCmd)
}
