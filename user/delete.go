package user

import (
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// UserDeleteCmd Deletes a User
var UserDeleteCmd = &cobra.Command{
	Use:   "user",
	Short: "Deletes a Passbolt User",
	Long:  `Deletes a Passbolt User`,
	RunE:  UserDelete,
}

func UserDelete(cmd *cobra.Command, args []string) error {
	resourceID, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}

	if resourceID == "" {
		return fmt.Errorf("No ID to Delete Provided")
	}

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	err = helper.DeleteUser(ctx, client, resourceID)
	if err != nil {
		return fmt.Errorf("Deleting User: %w", err)
	}
	return nil
}
