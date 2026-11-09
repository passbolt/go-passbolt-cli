package folder

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

// FolderDeleteCmd Deletes a Folder
var FolderDeleteCmd = makeFolderDeleteCmd()

func makeFolderDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "folder",
		Short: "Deletes a Passbolt Folder",
		Long:  `Deletes a Passbolt Folder`,
		RunE:  FolderDelete,
	}
}

func FolderDelete(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	if id == "" {
		return util.ErrNoID
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := client.DeleteFolder(ctx, id); err != nil {
			return fmt.Errorf("deleting Folder %s: %w", id, err)
		}
		return nil
	})
}
