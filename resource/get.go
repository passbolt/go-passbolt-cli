package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// ResourceGetCmd Gets a Passbolt Resource
var ResourceGetCmd = &cobra.Command{
	Use:   "resource",
	Short: "Gets a Passbolt Resource",
	Long:  `Gets a Passbolt Resource`,
	RunE:  ResourceGet,
}

// ResourcePermissionCmd Gets Permissions for Passbolt Resource
var ResourcePermissionCmd = &cobra.Command{
	Use:     "permission",
	Short:   "Gets Permissions for a Passbolt Resource",
	Long:    `Gets Permissions for a Passbolt Resource`,
	Aliases: []string{"permissions"},
	RunE:    ResourcePermission,
}

func init() {
	ResourceGetCmd.Flags().String("id", "", "id of Resource to Get")

	ResourceGetCmd.MarkFlagRequired("id")

	ResourceGetCmd.AddCommand(ResourcePermissionCmd)
	ResourcePermissionCmd.Flags().String("id", "", "id of Resource to Get")
	ResourcePermissionCmd.Flags().StringArrayP("column", "c", []string{"ID", "Aco", "AcoForeignKey", "Aro", "AroForeignKey", "Type"}, "Columns to return, possible Columns:\nID, Aco, AcoForeignKey, Aro, AroForeignKey, Type, CreatedTimestamp, ModifiedTimestamp")

}

func ResourceGet(cmd *cobra.Command, args []string) error {
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

	folderParentID, name, username, uri, password, description, err := helper.GetResource(
		ctx,
		client,
		id,
	)
	if err != nil {
		return fmt.Errorf("Getting Resource: %w", err)
	}

	if jsonOutput {
		jsonResource, err := json.MarshalIndent(ResourceJsonOutput{
			FolderParentID: &folderParentID,
			Name:           &name,
			Username:       &username,
			URI:            &uri,
			Password:       &password,
			Description:    &description,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonResource))
	} else {
		fmt.Printf("FolderParentID: %v\n", folderParentID)
		fmt.Printf("Name: %v\n", shellescape.StripUnsafe(name))
		fmt.Printf("Username: %v\n", shellescape.StripUnsafe(username))
		fmt.Printf("URI: %v\n", shellescape.StripUnsafe(uri))
		fmt.Printf("Password: %v\n", shellescape.StripUnsafe(password))
		fmt.Printf("Description: %v\n", shellescape.StripUnsafe(description))
	}
	return nil
}

func ResourcePermission(cmd *cobra.Command, args []string) error {
	resource, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return fmt.Errorf("You need to specify atleast one column to return")
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

	permissions, err := client.GetResourcePermissions(ctx, resource)
	if err != nil {
		return fmt.Errorf("Listing Permission: %w", err)
	}

	if jsonOutput {
		outputPermissions := []PermissionJsonOutput{}
		for i := range permissions {
			outputPermissions = append(outputPermissions, PermissionJsonOutput{
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
