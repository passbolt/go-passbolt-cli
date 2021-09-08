package folder

import (
	"context"
	"fmt"

	"github.com/speatzle/go-passbolt-cli/util"
	"github.com/speatzle/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// FolderGetCmd Gets a Passbolt Folder
var FolderGetCmd = &cobra.Command{
	Use:   "folder",
	Short: "Gets a Passbolt Folder",
	Long:  `Gets a Passbolt Folder`,
	RunE:  FolderGet,
}

func init() {
	FolderGetCmd.Flags().String("id", "", "id of Folder to Get")

	FolderGetCmd.MarkFlagRequired("id")
}

func FolderGet(cmd *cobra.Command, args []string) error {
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

	folderParentID, name, err := helper.GetFolder(
		ctx,
		client,
		id,
	)
	if err != nil {
		return fmt.Errorf("Getting Folder: %w", err)
	}
	fmt.Printf("FolderParentID: %v\n", folderParentID)
	fmt.Printf("Name: %v\n", name)
	return nil
}
