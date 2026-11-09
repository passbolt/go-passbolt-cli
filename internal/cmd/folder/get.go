package folder

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

// FolderGetCmd Gets a Passbolt Folder
var FolderGetCmd = makeFolderGetCmd()

func makeFolderGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "folder",
		Short: "Gets a Passbolt Folder",
		Long:  `Gets a Passbolt Folder`,
		RunE:  FolderGet,
	}
}

// FolderPermissionCmd Gets Permissions for Passbolt Folder
var FolderPermissionCmd = makeFolderPermissionCmd()

func makeFolderPermissionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "permission",
		Short:   "Gets Permissions for a Passbolt Folder",
		Long:    `Gets Permissions for a Passbolt Folder`,
		Aliases: []string{"permissions"},
		RunE:    FolderPermission,
	}
}

func configureGet() {
	FolderGetCmd.Flags().String("id", "", "id of Folder to Get")

	FolderGetCmd.MarkFlagRequired("id")

	FolderGetCmd.AddCommand(FolderPermissionCmd)
	FolderPermissionCmd.Flags().String("id", "", "id of Folder to get permissions for")
	FolderPermissionCmd.Flags().StringArrayP("column", "c", []string{"ID", "Aco", "AcoForeignKey", "Aro", "AroForeignKey", "Type"}, "Columns to return, possible Columns:\nID, Aco, AcoForeignKey, Aro, AroForeignKey, Type, CreatedTimestamp, ModifiedTimestamp")

	FolderPermissionCmd.MarkFlagRequired("id")
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

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		folder, err := client.GetFolder(ctx, id, nil)
		if err != nil {
			return fmt.Errorf("getting Folder: %w", err)
		}
		if jsonOutput {
			return util.PrintJSON(FolderJSONOutput{
				FolderParentID: &folder.FolderParentID,
				Name:           &folder.Name,
			})
		}
		util.Printf("FolderParentID: %v\n", util.Sanitize(folder.FolderParentID))
		util.Printf("Name: %v\n", util.Sanitize(folder.Name))
		return nil
	})
}

func FolderPermission(cmd *cobra.Command, args []string) error {
	folderID, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return util.NoColumnsError(util.PermissionColumns)
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		folder, err := client.GetFolder(ctx, folderID, &api.GetFolderOptions{
			ContainPermissions: true,
		})
		if err != nil {
			return fmt.Errorf("listing Permission for Folder %s: %w", folderID, err)
		}

		permissions := folder.Permissions

		if jsonOutput {
			return util.PrintJSON(util.PermissionsToJSONOutput(permissions))
		}

		if err := util.PrintPermissionTable(columns, permissions); err != nil {
			cmd.SilenceUsage = false
			return err
		}
		return nil
	})
}

func init() { configureGet() }
