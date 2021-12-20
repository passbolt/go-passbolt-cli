package user

import (
	"context"
	"fmt"

	"github.com/alessio/shellescape"
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

	role, username, firstname, lastname, err := helper.GetUser(
		ctx,
		client,
		id,
	)
	if err != nil {
		return fmt.Errorf("Getting User: %w", err)
	}
	fmt.Printf("Username: %v\n", shellescape.StripUnsafe(username))
	fmt.Printf("FirstName: %v\n", shellescape.StripUnsafe(firstname))
	fmt.Printf("LastName: %v\n", shellescape.StripUnsafe(lastname))
	fmt.Printf("Role: %v\n", shellescape.StripUnsafe(role))

	return nil
}
