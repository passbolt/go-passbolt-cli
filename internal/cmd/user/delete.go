package user

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// UserDeleteCmd Deletes a User
var UserDeleteCmd = makeUserDeleteCmd()

func makeUserDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "user",
		Short: "Deletes a Passbolt User",
		Long:  `Deletes a Passbolt User`,
		RunE:  UserDelete,
	}
}

func UserDelete(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	if id == "" {
		return util.ErrNoID
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := helper.DeleteUser(ctx, client, id); err != nil {
			return fmt.Errorf("deleting User %s: %w", id, err)
		}
		return nil
	})
}
