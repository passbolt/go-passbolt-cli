package resource

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// ResourceMoveCmd Moves a Passbolt Resource
var ResourceMoveCmd = makeResourceMoveCmd()

func makeResourceMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resource",
		Short: "Moves a Passbolt Resource into a Folder",
		Long:  `Moves a Passbolt Resource into a Folder`,
		RunE:  ResourceMove,
	}
}

func configureMove() {
	ResourceMoveCmd.Flags().String("id", "", "id of Resource to Move")
	ResourceMoveCmd.Flags().StringP("folderParentID", "f", "", "Folder in which to Move the Resource")

	ResourceMoveCmd.MarkFlagRequired("id")
	ResourceMoveCmd.MarkFlagRequired("folderParentID")
}

func ResourceMove(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	folderParentID, err := cmd.Flags().GetString("folderParentID")
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := helper.MoveResource(
			ctx,
			client,
			id,
			folderParentID,
		); err != nil {
			return fmt.Errorf("moving Resource: %w", err)
		}
		return nil
	})
}

func init() { configureMove() }
