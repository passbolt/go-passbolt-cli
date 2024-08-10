package permission

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/spf13/cobra"

	"github.com/pterm/pterm"
)

// PermissionListCmd Lists a Passbolt Permission
var PermissionListCmd = &cobra.Command{
	Use:     "permission",
	Short:   "Lists Passbolt Permissions",
	Long:    `Lists Passbolt Permissions`,
	Aliases: []string{"permissions"},
	RunE:    PermissionList,
}

func init() {
	PermissionListCmd.Flags().StringP("resource", "r", "", "Resource to list permissions")

	PermissionListCmd.Flags().StringArrayP("column", "c", []string{"ID", "Aco", "AcoForeignKey", "Aro", "AroForeignKey", "Type"}, "Columns to return, possible Columns:\nID, Aco, AcoForeignKey, Aro, AroForeignKey, Type, CreatedTimestamp, ModifiedTimestamp")

	PermissionListCmd.MarkFlagRequired("resource")
}

func PermissionList(cmd *cobra.Command, args []string) error {
	resource, err := cmd.Flags().GetString("resource")
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
