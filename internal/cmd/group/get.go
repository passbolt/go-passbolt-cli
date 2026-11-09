package group

import (
	"context"
	"fmt"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// GroupGetCmd Gets a Passbolt Group
var GroupGetCmd = makeGroupGetCmd()

func makeGroupGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "group",
		Short: "Gets a Passbolt Group",
		Long:  `Gets a Passbolt Group`,
		RunE:  GroupGet,
	}
}

func configureGet() {
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

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		name, memberships, err := helper.GetGroup(
			ctx,
			client,
			id,
		)
		if err != nil {
			return fmt.Errorf("getting Group: %w", err)
		}

		if jsonOutput {
			groupUserMemberships := []GroupUserMembershipJSONOutput{}
			for i := range memberships {
				groupUserMemberships = append(groupUserMemberships, GroupUserMembershipJSONOutput{
					ID:             &memberships[i].UserID,
					Username:       &memberships[i].Username,
					FirstName:      &memberships[i].UserFirstName,
					LastName:       &memberships[i].UserLastName,
					IsGroupManager: &memberships[i].IsGroupManager,
				})
			}

			return util.PrintJSON(GroupJSONOutput{
				Name:  &name,
				Users: groupUserMemberships,
			})
		}

		util.Printf("Name: %v\n", util.Sanitize(name))
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
						entry[i] = membership.Username
					case "userfirstname":
						entry[i] = membership.UserFirstName
					case "userlastname":
						entry[i] = membership.UserLastName
					default:
						cmd.SilenceUsage = false
						return fmt.Errorf("unknown Column: %v", columns[i])
					}
				}
				data = append(data, entry)
			}

			util.RenderTable(data)
		}
		return nil
	})
}

func init() { configureGet() }
