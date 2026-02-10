package resource

import (
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/spf13/cobra"
)

// ResourceDeleteCmd Deletes a Resource
var ResourceDeleteCmd = &cobra.Command{
	Use:   "resource",
	Short: "Deletes a Passbolt Resource",
	Long:  `Deletes a Passbolt Resource`,
	RunE:  ResourceDelete,
}

func ResourceDelete(cmd *cobra.Command, args []string) error {
	resourceID, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}

	if resourceID == "" {
		return fmt.Errorf("No ID to Delete Provided")
	}

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	err = client.DeleteResource(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("Deleting Resource: %w", err)
	}
	return nil
}
