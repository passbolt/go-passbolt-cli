package group

import (
	"context"
	"fmt"
	"strings"

	"github.com/speatzle/go-passbolt-cli/util"
	"github.com/speatzle/go-passbolt/api"
	"github.com/spf13/cobra"

	"github.com/pterm/pterm"
)

// GroupListCmd Lists a Passbolt Group
var GroupListCmd = &cobra.Command{
	Use:     "group",
	Short:   "Lists Passbolt Groups",
	Long:    `Lists Passbolt Groups`,
	Aliases: []string{"groups"},
	RunE:    GroupList,
}

func init() {
	GroupListCmd.Flags().StringArrayP("users", "u", []string{}, "Groups that are shared with group")
	GroupListCmd.Flags().StringArrayP("managers", "m", []string{}, "Groups that are in folder")

	GroupListCmd.Flags().StringArrayP("columns", "c", []string{"ID", "Name"}, "Columns to return, possible Columns:\nID, Name")
}

func GroupList(cmd *cobra.Command, args []string) error {
	users, err := cmd.Flags().GetStringArray("users")
	if err != nil {
		return err
	}
	managers, err := cmd.Flags().GetStringArray("managers")
	if err != nil {
		return err
	}
	columns, err := cmd.Flags().GetStringArray("columns")
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return fmt.Errorf("You need to specify atleast one column to return")
	}

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	resources, err := client.GetGroups(ctx, &api.GetGroupsOptions{
		FilterHasUsers:    users,
		FilterHasManagers: managers,
	})
	if err != nil {
		return fmt.Errorf("Listing Group: %w", err)
	}

	data := pterm.TableData{columns}

	for _, resource := range resources {
		entry := make([]string, len(columns))
		for i := range columns {
			switch strings.ToLower(columns[i]) {
			case "id":
				entry[i] = resource.ID
			case "name":
				entry[i] = resource.Name
			default:
				cmd.SilenceUsage = false
				return fmt.Errorf("Unknown Column: %v", columns[i])
			}
		}
		data = append(data, entry)
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	return nil
}
