package folder

import (
	"context"
	"encoding/json"
	"fmt"

	"al.essio.dev/pkg/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
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
	jsonOutput, err := cmd.Flags().GetBool("json")
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

	folder, err := client.GetFolder(ctx, id, nil)
	if err != nil {
		return fmt.Errorf("Getting Folder: %w", err)
	}
	if jsonOutput {
		jsonGroup, err := json.MarshalIndent(FolderJsonOutput{
			FolderParentID: &folder.FolderParentID,
			Name:           &folder.Name,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonGroup))
	} else {
		fmt.Printf("FolderParentID: %v\n", folder.FolderParentID)
		fmt.Printf("Name: %v\n", shellescape.StripUnsafe(folder.Name))
	}
	return nil
}
