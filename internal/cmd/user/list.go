package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

// UserListCmd Lists a Passbolt User
var UserListCmd = &cobra.Command{
	Use:     "user",
	Short:   "Lists Passbolt Users",
	Long:    `Lists Passbolt Users`,
	Aliases: []string{"users"},
	RunE:    UserList,
}

func init() {
	flags := UserListCmd.Flags()
	flags.StringArrayP("group", "g", []string{}, "Users that are members of all of these groups")
	flags.StringArrayP("resource", "r", []string{}, "Users that have access to resources")
	flags.StringP("search", "s", "", "Search for Users")
	flags.BoolP("admin", "a", false, "Only show Admins")
	flags.StringArrayP("column", "c", userColumns.DefaultTableColumns(), "Columns to return (default list only for table format; JSON format includes all fields by default).\nPossible Columns: "+strings.Join(userColumns.Resolver().Canonical(), ", ")+"\nLegacy PascalCase column names (ID, FirstName, ...) remain accepted for backwards compatibility.")
}

type userListConfig struct {
	groups         []string
	resources      []string
	search         string
	admin          bool
	columns        []string
	columnsChanged bool
	jsonOutput     bool
	celFilter      string
}

func UserList(cmd *cobra.Command, args []string) error {
	config, err := parseUserListFlags(cmd)
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		users, err := client.GetUsers(ctx, &api.GetUsersOptions{
			FilterHasGroup:  config.groups,
			FilterHasAccess: config.resources,
			FilterSearch:    config.search,
			FilterIsAdmin:   config.admin,
		})
		if err != nil {
			return fmt.Errorf("listing User: %w", err)
		}

		users, err = userColumns.Filter(ctx, users, config.celFilter)
		if err != nil {
			return err
		}

		if config.jsonOutput {
			return printJSONUsers(users, config.columnsChanged, config.columns)
		}

		return printTableUsers(config.columns, users)
	})
}

func printJSONUsers(users []api.User, isColumnsChanged bool, columns []string) error {
	outputUsers := make([]UserJSONOutput, len(users))
	for i := range users {
		outputUsers[i] = userJSONOutput(users[i])
	}

	if isColumnsChanged {
		return util.PrintJSONColumnFiltered(outputUsers, columns)
	}

	return util.PrintJSON(outputUsers)
}

func printTableUsers(columns []string, users []api.User) error {
	// Input is normalized by parseUserListFlags; a miss in the resolver is a
	// defensive guard against a future caller that skips that step.
	return util.PrintTable(columns, users, userColumns.TableValue)
}

func parseUserListFlags(cmd *cobra.Command) (*userListConfig, error) {
	groups, err := cmd.Flags().GetStringArray("group")
	if err != nil {
		return nil, err
	}
	resources, err := cmd.Flags().GetStringArray("resource")
	if err != nil {
		return nil, err
	}
	search, err := cmd.Flags().GetString("search")
	if err != nil {
		return nil, err
	}
	admin, err := cmd.Flags().GetBool("admin")
	if err != nil {
		return nil, err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, util.NoColumnsError(userColumns.Resolver().Canonical())
	}
	columns, err = userColumns.Resolver().NormalizeAll(columns)
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

	return &userListConfig{
		groups:         groups,
		resources:      resources,
		search:         search,
		admin:          admin,
		columns:        columns,
		columnsChanged: cmd.Flags().Changed("column"),
		jsonOutput:     jsonOutput,
		celFilter:      celFilter,
	}, nil
}
