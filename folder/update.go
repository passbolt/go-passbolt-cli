package folder

import (
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// FolderUpdateCmd Updates a Passbolt Folder
var FolderUpdateCmd = &cobra.Command{
	Use:   "resource",
	Short: "Updates a Passbolt Folder",
	Long:  `Updates a Passbolt Folder`,
	RunE:  FolderUpdate,
}

func init() {
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

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	err = helper.UpdateFolder(
		ctx,
		client,
		id,
		name,
	)
	if err != nil {
		return fmt.Errorf("Updating Folder: %w", err)
	}
	return nil
}
