package cmd

import (
	"github.com/spf13/cobra"
)

// moveCmd represents the move command
var moveCmd = &cobra.Command{
	Use:   "move",
	Short: "Moves a Passbolt Entity",
	Long:  `Moves a Passbolt Entity`,
}

func init() {
}
