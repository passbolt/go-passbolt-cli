package group

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"al.essio.dev/pkg/shellescape"
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

	ctx, cancel := util.GetContext()
	defer cancel()

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
		groupUserMemberships := []GroupUserMembershipJsonOutput{}
		for i := range memberships {
			groupUserMemberships = append(groupUserMemberships, GroupUserMembershipJsonOutput{
				ID:             &memberships[i].UserID,
				Username:       &memberships[i].Username,
				FirstName:      &memberships[i].UserFirstName,
				LastName:       &memberships[i].UserLastName,
				IsGroupManager: &memberships[i].IsGroupManager,
			})
		}

		jsonGroup, err := json.MarshalIndent(GroupJsonOutput{
			Name:  &name,
			Users: groupUserMemberships,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonGroup))

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
