package user

import (
	"context"
	"fmt"

	"github.com/speatzle/go-passbolt-cli/util"
	"github.com/speatzle/go-passbolt/helper"
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

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	err = helper.UpdateUser(
		ctx,
		client,
		id,
		role,
		firstname,
		lastname,
	)
	if err != nil {
		return fmt.Errorf("Updating User: %w", err)
	}
	return nil
}
