package group

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// GroupGetCmd Gets a Passbolt Group
var GroupGetCmd = &cobra.Command{
	Use:   "group",
	Short: "Gets a Passbolt Group",
	Long:  `Gets a Passbolt Group`,
	RunE:  GroupGet,
}

func init() {
	GroupGetCmd.Flags().String("id", "", "id of Group to Get")

	GroupGetCmd.Flags().StringArrayP("column", "c", []string{"UserID", "Username", "UserFirstName", "UserLastName", "IsGroupManager"}, "Membership Columns to return, possible Columns:\nUserID, Username, UserFirstName, UserLastName, IsGroupManager")

	GroupGetCmd.MarkFlagRequired("id")
}

func GroupGet(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	columns, err := cmd.Flags().GetStringArray("column")
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

	name, memberships, err := helper.GetGroup(
		ctx,
		client,
		id,
	)
	if err != nil {
		return fmt.Errorf("Getting Group: %w", err)
	}

	if jsonOutput {
		group, err := client.GetGroup(ctx, id)
		if err != nil {
			return err
		}
		jsonGroup, err := json.MarshalIndent(group, "", "  ")
		if err != nil {
			return err
		}

		var tempMap map[string]interface{}
		if err := json.Unmarshal(jsonGroup, &tempMap); err != nil {
			return err
		}
		tempMap["group_users"] = memberships
		jsonResult, err := json.MarshalIndent(tempMap, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(jsonResult))
	} else {
		fmt.Printf("Name: %v\n", name)
		// Print Memberships
		if len(columns) != 0 {
			data := pterm.TableData{columns}

			for _, membership := range memberships {
				entry := make([]string, len(columns))
				for i := range columns {
					switch strings.ToLower(columns[i]) {
					case "userid":
						entry[i] = membership.UserID
					case "isgroupmanager":
						entry[i] = fmt.Sprint(membership.IsGroupManager)
					case "username":
						entry[i] = shellescape.StripUnsafe(membership.Username)
					case "userfirstname":
						entry[i] = shellescape.StripUnsafe(membership.UserFirstName)
					case "userlastname":
						entry[i] = shellescape.StripUnsafe(membership.UserLastName)
					default:
						cmd.SilenceUsage = false
						return fmt.Errorf("Unknown Column: %v", columns[i])
					}
				}
				data = append(data, entry)
			}

			pterm.DefaultTable.WithHasHeader().WithData(data).Render()
		}
	}
	return nil
}
