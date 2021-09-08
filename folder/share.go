package folder

import (
	"context"
	"fmt"

	"github.com/speatzle/go-passbolt-cli/util"
	"github.com/speatzle/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// FolderShareCmd Shares a Passbolt Folder
var FolderShareCmd = &cobra.Command{
	Use:   "folder",
	Short: "Shares a Passbolt Folder",
	Long:  `Shares a Passbolt Folder`,
	RunE:  FolderShare,
}

func init() {
	FolderShareCmd.Flags().String("id", "", "id of Folder to Share")
	FolderShareCmd.Flags().IntP("type", "t", 1, "Permission Type (1 Read Only, 7 Can Update, 15 Owner)")
	FolderShareCmd.Flags().StringArrayP("users", "u", []string{}, "User id's to share with")
	FolderShareCmd.Flags().StringArrayP("groups", "g", []string{}, "Group id's to share with")

	FolderShareCmd.MarkFlagRequired("id")
	FolderShareCmd.MarkFlagRequired("type")
}

func FolderShare(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	pType, err := cmd.Flags().GetInt("type")
	if err != nil {
		return err
	}
	users, err := cmd.Flags().GetStringArray("users")
	if err != nil {
		return err
	}
	groups, err := cmd.Flags().GetStringArray("groups")
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

	err = helper.ShareFolderWithUsersAndGroups(
		ctx,
		client,
		id,
		users,
		groups,
		pType,
	)
	if err != nil {
		return fmt.Errorf("Sharing Folder: %w", err)
	}
	return nil
}
