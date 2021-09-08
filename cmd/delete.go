package cmd

import (
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Deletes a Passbolt Entity",
	Long:    `Deletes a Passbolt Entity`,
	Aliases: []string{"remove"},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.PersistentFlags().String("id", "", "ID of the Entity to Delete")
}
