package group

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/spf13/cobra"
)

// GroupDeleteCmd Deletes a Group
var GroupDeleteCmd = &cobra.Command{
	Use:   "group",
	Short: "Deletes a Passbolt Group",
	Long:  `Deletes a Passbolt Group`,
	RunE:  GroupDelete,
}

func GroupDelete(cmd *cobra.Command, args []string) error {
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
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	err = client.DeleteGroup(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("Deleting Group: %w", err)
	}
	return nil
}
