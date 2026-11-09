package folder

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// FolderShareCmd Shares a Passbolt Folder
var FolderShareCmd = makeFolderShareCmd()

func makeFolderShareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "folder",
		Short: "Shares a Passbolt Folder",
		Long:  `Shares a Passbolt Folder`,
		RunE:  FolderShare,
	}
}

func configureShare() {
	FolderShareCmd.Flags().String("id", "", "id of Folder to Share")
	FolderShareCmd.Flags().IntP("type", "t", 1, "Permission Type (1 Read Only, 7 Can Update, 15 Owner)")
	FolderShareCmd.Flags().StringArrayP("user", "u", []string{}, "User id's to share with")
	FolderShareCmd.Flags().StringArrayP("group", "g", []string{}, "Group id's to share with")

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
	users, err := cmd.Flags().GetStringArray("user")
	if err != nil {
		return err
	}
	groups, err := cmd.Flags().GetStringArray("group")
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := helper.ShareFolderWithUsersAndGroups(
			ctx,
			client,
			id,
			users,
			groups,
			pType,
		); err != nil {
			return fmt.Errorf("sharing Folder: %w", err)
		}
		return nil
	})
}

func init() { configureShare() }
