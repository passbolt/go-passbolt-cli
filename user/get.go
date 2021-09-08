package user

import (
	"context"
	"fmt"

	"github.com/speatzle/go-passbolt-cli/util"
	"github.com/speatzle/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// UserGetCmd Gets a Passbolt User
var UserGetCmd = &cobra.Command{
	Use:   "user",
	Short: "Gets a Passbolt User",
	Long:  `Gets a Passbolt User`,
	RunE:  UserGet,
}

func init() {
	UserGetCmd.Flags().String("id", "", "id of User to Get")

	UserGetCmd.MarkFlagRequired("id")
}

func UserGet(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
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

	username, firstname, lastname, role, err := helper.GetUser(
		ctx,
		client,
		id,
	)
	if err != nil {
		return fmt.Errorf("Getting User: %w", err)
	}
	fmt.Printf("Username: %v\n", username)
	fmt.Printf("FirstName: %v\n", firstname)
	fmt.Printf("LastName: %v\n", lastname)
	fmt.Printf("Role: %v\n", role)

	return nil
}
