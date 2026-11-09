package folder

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// FolderUpdateCmd Updates a Passbolt Folder
var FolderUpdateCmd = makeFolderUpdateCmd()

func makeFolderUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "folder",
		Short: "Updates a Passbolt Folder",
		Long:  `Updates a Passbolt Folder`,
		RunE:  FolderUpdate,
	}
}

func configureUpdate() {
	FolderUpdateCmd.Flags().String("id", "", "id of Folder to Update")
	FolderUpdateCmd.Flags().StringP("name", "n", "", "Folder Name")

	FolderUpdateCmd.MarkFlagRequired("id")
	FolderUpdateCmd.MarkFlagRequired("name")
}

func FolderUpdate(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := helper.UpdateFolder(
			ctx,
			client,
			id,
			name,
		); err != nil {
			return fmt.Errorf("updating Folder: %w", err)
		}
		return nil
	})
}

func init() { configureUpdate() }
