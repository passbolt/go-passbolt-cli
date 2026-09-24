package folder

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
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
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		id, err := helper.CreateFolder(
			ctx,
			client,
			folderParentID,
			name,
		)
		if err != nil {
			return fmt.Errorf("creating Folder: %w", err)
		}

		if jsonOutput {
			return util.PrintJSON(map[string]string{"id": id})
		}
		util.Printf("FolderID: %v\n", util.Sanitize(id))
		return nil
	})
}
