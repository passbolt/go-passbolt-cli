package folder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
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

	ctx, cancel := util.GetContext()
	defer cancel()

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

	if jsonOutput {
		jsonId, err := json.MarshalIndent(
			map[string]string{"id": id},
			"",
			"  ",
		)
		if err != nil {
			return fmt.Errorf("Marshalling Json: %w", err)
		}
		fmt.Println(string(jsonId))
	} else {
		fmt.Printf("FolderID: %v\n", id)
	}
	return nil
}
