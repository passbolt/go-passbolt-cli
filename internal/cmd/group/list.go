package group

import (
	"context"
	"fmt"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
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
	flags := GroupListCmd.Flags()
	flags.StringArrayP("user", "u", []string{}, "Groups that have all of these users as members")
	flags.StringArrayP("manager", "m", []string{}, "Groups that have all of these users as managers")
	flags.StringArrayP("column", "c", groupColumns.DefaultTableColumns(), "Columns to return (default list only for table format; JSON format includes all fields by default).\nPossible Columns: "+strings.Join(groupColumns.Resolver().Canonical(), ", ")+"\nLegacy PascalCase column names (ID, Name, CreatedTimestamp, ...) remain accepted for backwards compatibility.")
}

type groupListConfig struct {
	users          []string
	managers       []string
	columns        []string
	columnsChanged bool
	jsonOutput     bool
	celFilter      string
}

func GroupList(cmd *cobra.Command, args []string) error {
	config, err := parseGroupListFlags(cmd)
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		groups, err := client.GetGroups(ctx, &api.GetGroupsOptions{
			FilterHasUsers:    config.users,
			FilterHasManagers: config.managers,
		})
		if err != nil {
			return fmt.Errorf("listing Group: %w", err)
		}

		groups, err = groupColumns.Filter(ctx, groups, config.celFilter)
		if err != nil {
			return err
		}

		if config.jsonOutput {
			return printJSONGroups(groups, config.columnsChanged, config.columns)
		}

		return printTableGroups(config.columns, groups)
	})
}

func printJSONGroups(groups []api.Group, isColumnsChanged bool, columns []string) error {
	outputGroups := make([]GroupJSONOutput, len(groups))
	for i := range groups {
		outputGroups[i] = GroupJSONOutput{
			ID:                &groups[i].ID,
			Name:              &groups[i].Name,
			CreatedTimestamp:  &groups[i].Created.Time,
			ModifiedTimestamp: &groups[i].Modified.Time,
			Deleted:           groups[i].Deleted,
			UserCount:         groups[i].UserCount,
		}
	}

	if isColumnsChanged {
		return util.PrintJSONColumnFiltered(outputGroups, columns)
	}

	return util.PrintJSON(outputGroups)
}

func printTableGroups(columns []string, groups []api.Group) error {
	// Input is normalized by parseGroupListFlags; a miss in the resolver is a
	// defensive guard against a future caller that skips that step.
	return util.PrintTable(columns, groups, groupColumns.TableValue)
}

func parseGroupListFlags(cmd *cobra.Command) (*groupListConfig, error) {
	users, err := cmd.Flags().GetStringArray("user")
	if err != nil {
		return nil, err
	}
	managers, err := cmd.Flags().GetStringArray("manager")
	if err != nil {
		return nil, err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, util.NoColumnsError(groupColumns.Resolver().Canonical())
	}
	columns, err = groupColumns.Resolver().NormalizeAll(columns)
	if err != nil {
		return nil, err
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return nil, err
	}
	celFilter, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return &groupListConfig{
		users:          users,
		managers:       managers,
		columns:        columns,
		columnsChanged: cmd.Flags().Changed("column"),
		jsonOutput:     jsonOutput,
		celFilter:      celFilter,
	}, nil
}
