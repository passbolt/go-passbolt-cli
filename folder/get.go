package folder

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"al.essio.dev/pkg/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// FolderGetCmd Gets a Passbolt Folder
var FolderGetCmd = &cobra.Command{
	Use:   "folder",
	Short: "Gets a Passbolt Folder",
	Long:  `Gets a Passbolt Folder`,
	RunE:  FolderGet,
}

// FolderPermissionCmd Gets Permissions for Passbolt Folder
var FolderPermissionCmd = &cobra.Command{
	Use:     "permission",
	Short:   "Gets Permissions for a Passbolt Folder",
	Long:    `Gets Permissions for a Passbolt Folder`,
	Aliases: []string{"permissions"},
	RunE:    FolderPermission,
}

func init() {
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

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
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
		return fmt.Errorf("You need to specify at least one column to return")
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
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	folder, err := client.GetFolder(ctx, folderID, &api.GetFolderOptions{
		ContainPermissions: true,
	})
	if err != nil {
		return fmt.Errorf("Listing Permission: %w", err)
	}

	permissions := folder.Permissions

	if jsonOutput {
		outputPermissions := []util.PermissionJsonOutput{}
		for i := range permissions {
			outputPermissions = append(outputPermissions, util.PermissionJsonOutput{
				ID:                &permissions[i].ID,
				Aco:               &permissions[i].ACO,
				AcoForeignKey:     &permissions[i].ACOForeignKey,
				Aro:               &permissions[i].ARO,
				AroForeignKey:     &permissions[i].AROForeignKey,
				Type:              &permissions[i].Type,
				CreatedTimestamp:  &permissions[i].Created.Time,
				ModifiedTimestamp: &permissions[i].Modified.Time,
			})
		}
		jsonPermissions, err := json.MarshalIndent(outputPermissions, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonPermissions))
	} else {
		data := pterm.TableData{columns}

		for _, permission := range permissions {
			entry := make([]string, len(columns))
			for i := range columns {
				switch strings.ToLower(columns[i]) {
				case "id":
					entry[i] = permission.ID
				case "aco":
					entry[i] = permission.ACO
				case "acoforeignkey":
					entry[i] = permission.ACOForeignKey
				case "aro":
					entry[i] = permission.ARO
				case "aroforeignkey":
					entry[i] = permission.AROForeignKey
				case "type":
					entry[i] = strconv.Itoa(permission.Type)
				case "createdtimestamp":
					entry[i] = permission.Created.Format(time.RFC3339)
				case "modifiedtimestamp":
					entry[i] = permission.Modified.Format(time.RFC3339)
				default:
					cmd.SilenceUsage = false
					return fmt.Errorf("Unknown Column: %v", columns[i])
				}
			}
			data = append(data, entry)
		}

		pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	}

	return nil
}
