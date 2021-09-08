package folder

import (
	"context"
	"fmt"

	"github.com/speatzle/go-passbolt-cli/util"
	"github.com/speatzle/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// FolderCreateCmd Creates a Passbolt Folder
var FolderCreateCmd = &cobra.Command{
	Use:   "folder",
	Short: "Creates a Passbolt Folder",
	Long:  `Creates a Passbolt Folder and Returns the Folders ID`,
	RunE:  FolderCreate,
}

func init() {
	FolderCreateCmd.Flags().StringP("name", "n", "", "Folder Name")
	FolderCreateCmd.Flags().StringP("folderParentID", "f", "", "Folder in which to create the Folder")

	FolderCreateCmd.MarkFlagRequired("name")
}

func FolderCreate(cmd *cobra.Command, args []string) error {
	folderParentID, err := cmd.Flags().GetString("folderParentID")
	if err != nil {
		return err
	}
	name, err := cmd.Flags().GetString("name")
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

	id, err := helper.CreateFolder(
		ctx,
		client,
		folderParentID,
		name,
	)
	if err != nil {
		return fmt.Errorf("Creating Folder: %w", err)
	}

	fmt.Printf("FolderID: %v\n", id)
	return nil
}
