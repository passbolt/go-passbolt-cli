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
var UserUpdateCmd = &cobra.Command{
	Use:   "user",
	Short: "Updates a Passbolt User",
	Long:  `Updates a Passbolt User`,
	RunE:  UserUpdate,
}

func init() {
	UserUpdateCmd.Flags().String("id", "", "id of User to Update")
	UserUpdateCmd.Flags().StringP("firstname", "f", "", "User FirstName")
	UserUpdateCmd.Flags().StringP("lastname", "l", "", "User LastName")
	UserUpdateCmd.Flags().StringP("role", "r", "", "User Role")
	UserUpdateCmd.Flags().Bool("disabled", false, "Disable the account (use --disabled=false to enable it); cannot be combined with profile or role changes")

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

	stateChanged := cmd.Flags().Changed("disabled")
	disabled := false
	if stateChanged {
		disabled, err = cmd.Flags().GetBool("disabled")
		if err != nil {
			return err
		}
		for _, flag := range []string{"firstname", "lastname", "role"} {
			if cmd.Flags().Changed(flag) {
				return fmt.Errorf("cannot combine --disabled with --%s", flag)
			}
		}
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if stateChanged {
			return updateUserState(ctx, client, id, disabled)
		}
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

func updateUserState(ctx context.Context, client *api.Client, id string, disabled bool) error {
	var err error
	if disabled {
		err = client.DisableUser(ctx, id)
	} else {
		err = client.EnableUser(ctx, id)
	}
	if err != nil {
		return fmt.Errorf("updating user account state: %w", err)
	}
	return nil
}
