package folder

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// FolderMoveCmd Moves a Passbolt Folder
var FolderMoveCmd = makeFolderMoveCmd()

func makeFolderMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "folder",
		Short: "Moves a Passbolt Folder into a Folder",
		Long:  `Moves a Passbolt Folder into a Folder`,
		RunE:  FolderMove,
	}
}

func configureMove() {
	FolderMoveCmd.Flags().String("id", "", "id of Folder to Move")
	FolderMoveCmd.Flags().StringP("folderParentID", "f", "", "Folder in which to Move the Folder")

	FolderMoveCmd.MarkFlagRequired("id")
	FolderMoveCmd.MarkFlagRequired("folderParentID")
}

func FolderMove(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	folderParentID, err := cmd.Flags().GetString("folderParentID")
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := helper.MoveFolder(
			ctx,
			client,
			id,
			folderParentID,
		); err != nil {
			return fmt.Errorf("moving Folder: %w", err)
		}
		return nil
	})
}

func init() { configureMove() }
