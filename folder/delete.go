package folder

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/spf13/cobra"
)

// FolderDeleteCmd Deletes a Folder
var FolderDeleteCmd = &cobra.Command{
	Use:   "folder",
	Short: "Deletes a Passbolt Folder",
	Long:  `Deletes a Passbolt Folder`,
	RunE:  FolderDelete,
}

func FolderDelete(cmd *cobra.Command, args []string) error {
	folderID, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}

	if folderID == "" {
		return fmt.Errorf("No ID to Delete Provided")
	}

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	client.DeleteFolder(ctx, folderID)
	if err != nil {
		return fmt.Errorf("Deleting Folder: %w", err)
	}
	return nil
}
