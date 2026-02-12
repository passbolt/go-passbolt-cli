package resource

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// ResourceMoveCmd Moves a Passbolt Resource
var ResourceMoveCmd = &cobra.Command{
	Use:   "resource",
	Short: "Moves a Passbolt Resource into a Folder",
	Long:  `Moves a Passbolt Resource into a Folder`,
	RunE:  ResourceMove,
}

func init() {
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

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	err = helper.MoveResource(
		ctx,
		client,
		id,
		folderParentID,
	)
	if err != nil {
		return fmt.Errorf("Moving Resource: %w", err)
	}
	return nil
}
