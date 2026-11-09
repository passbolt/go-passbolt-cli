package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/user"
	"github.com/spf13/cobra"
)

var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Requests account recovery",
}

func init() {
	rootCmd.AddCommand(recoverCmd)
	recoverCmd.AddCommand(user.UserRecoverCmd)
}
