package user

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// UserUpdateCmd Updates a Passbolt User
var UserUpdateCmd = makeUserUpdateCmd()

func makeUserUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "user",
		Short: "Updates a Passbolt User",
		Long:  `Updates a Passbolt User`,
		RunE:  UserUpdate,
	}
}

func configureUpdate() {
	UserUpdateCmd.Flags().String("id", "", "id of User to Update")
	UserUpdateCmd.Flags().StringP("firstname", "f", "", "User FirstName")
	UserUpdateCmd.Flags().StringP("lastname", "l", "", "User LastName")
	UserUpdateCmd.Flags().StringP("role", "r", "", "User Role")

	UserUpdateCmd.MarkFlagRequired("id")
}

func UserUpdate(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	firstname, err := cmd.Flags().GetString("firstname")
	if err != nil {
		return err
	}
	lastname, err := cmd.Flags().GetString("lastname")
	if err != nil {
		return err
	}
	role, err := cmd.Flags().GetString("role")
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := helper.UpdateUser(
			ctx,
			client,
			id,
			role,
			firstname,
			lastname,
		); err != nil {
			return fmt.Errorf("updating User: %w", err)
		}
		return nil
	})
}

func init() { configureUpdate() }
