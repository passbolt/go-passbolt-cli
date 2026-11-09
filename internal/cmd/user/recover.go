package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

// UserRecoverCmd requests recovery mail or renews an unfinished account's invitation.
var UserRecoverCmd = &cobra.Command{
	Use:   "user",
	Short: "Requests account recovery mail or renews an invitation",
	Long:  "Requests account recovery mail, or renews the invitation for an account that has not completed setup. Success does not confirm account existence or email delivery.",
	Args:  cobra.NoArgs,
	RunE:  UserRecover,
}

func init() {
	UserRecoverCmd.Flags().StringP("username", "u", "", "Email address of the user")
	UserRecoverCmd.MarkFlagRequired("username")
}

func UserRecover(cmd *cobra.Command, args []string) error {
	username, err := cmd.Flags().GetString("username")
	if err != nil {
		return err
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username must not be empty")
	}
	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		return recoverUser(ctx, client, username)
	})
}

func recoverUser(ctx context.Context, client *api.Client, username string) error {
	if err := client.RecoverUser(ctx, username); err != nil {
		return fmt.Errorf("requesting user recovery: %w", err)
	}
	return nil
}
