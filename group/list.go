package group

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
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
	GroupListCmd.Flags().String("filter", "", "Filtercriteria on group name by a regular expression.\nExample:--filter '.*somegroup?'")
	GroupListCmd.Flags().StringArrayP("user", "u", []string{}, "Groups that are shared with group")
	GroupListCmd.Flags().StringArrayP("manager", "m", []string{}, "Groups that are in folder")

	GroupListCmd.Flags().StringArrayP("column", "c", []string{"ID", "Name"}, "Columns to return, possible Columns:\nID, Name, CreatedTimestamp, ModifiedTimestamp")
}

func GroupList(cmd *cobra.Command, args []string) error {
	users, err := cmd.Flags().GetStringArray("user")
	if err != nil {
		return err
	}
	managers, err := cmd.Flags().GetStringArray("manager")
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
	filter, err := cmd.Flags().GetString("filter")
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

	groups, err := client.GetGroups(ctx, &api.GetGroupsOptions{
		FilterHasUsers:    users,
		FilterHasManagers: managers,
	})
	if err != nil {
		return fmt.Errorf("Listing Group: %w", err)
	}

	filteredGroups, err := filteredGroups(&groups, filter)
	if err != nil {
		return fmt.Errorf("Listing filtered Groups: %w", err)
	}

	if jsonOutput {
		outputGroups := []GroupJsonOutput{}
		for i := range *filteredGroups {
			outputGroups = append(outputGroups, GroupJsonOutput{
				ID:                &groups[i].ID,
				Name:              &groups[i].Name,
				CreatedTimestamp:  &groups[i].Created.Time,
				ModifiedTimestamp: &groups[i].Modified.Time,
			})
		}
		jsonGroups, err := json.MarshalIndent(outputGroups, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonGroups))
	} else {
		data := pterm.TableData{columns}

		for _, group := range *filteredGroups {
			entry := make([]string, len(columns))
			for i := range columns {
				switch strings.ToLower(columns[i]) {
				case "id":
					entry[i] = group.ID
				case "name":
					entry[i] = shellescape.StripUnsafe(group.Name)
				case "createdtimestamp":
					entry[i] = group.Created.Format(time.RFC3339)
				case "modifiedtimestamp":
					entry[i] = group.Modified.Format(time.RFC3339)
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

func filteredGroups(groups *[]api.Group, filter string) (*[]api.Group, error) {
	if filter == "" {
		return groups, nil
	}

	filteredGroups := []api.Group{}

	for _, group := range *groups {

		matches, err := regexp.MatchString(filter, group.Name)
		if err != nil {
			return nil, err
		}
		if matches {
			filteredGroups = append(filteredGroups, group)
		}
	}

	return &filteredGroups, nil
}
