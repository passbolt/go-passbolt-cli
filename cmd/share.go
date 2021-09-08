package cmd

import (
	"github.com/spf13/cobra"
)

// shareCmd represents the share command
var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Shares a Passbolt Entity",
	Long:  `Shares a Passbolt Entity`,
}

func init() {
	rootCmd.AddCommand(shareCmd)
}
