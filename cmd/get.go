package cmd

import (
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

}
