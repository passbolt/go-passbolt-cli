package user

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"al.essio.dev/pkg/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"

	"github.com/pterm/pterm"
)

var defaultTableColumns = []string{"ID", "Username", "FirstName", "LastName", "Role"}

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
	flags.StringArrayP("group", "g", []string{}, "Users that are members of groups")
	flags.StringArrayP("resource", "r", []string{}, "Users that have access to resources")
	flags.StringP("search", "s", "", "Search for Users")
	flags.BoolP("admin", "a", false, "Only show Admins")
	flags.StringArrayP("column", "c", defaultTableColumns, "Columns to return (default list only for table format; JSON format includes all fields by default).\nPossible Columns: ID, Username, FirstName, LastName, Role, CreatedTimestamp, ModifiedTimestamp")
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

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	users, err := client.GetUsers(ctx, &api.GetUsersOptions{
		FilterHasGroup:  config.groups,
		FilterHasAccess: config.resources,
		FilterSearch:    config.search,
		FilterIsAdmin:   config.admin,
	})
	if err != nil {
		return fmt.Errorf("Listing User: %w", err)
	}

	users, err = filterUsers(&users, config.celFilter, ctx)
	if err != nil {
		return err
	}

	if config.jsonOutput {
		return printJsonUsers(users, config.columnsChanged, config.columns)
	}

	return printTableUsers(config.columns, users)
}

func printJsonUsers(users []api.User, isColumnsChanged bool, columns []string) error {
	outputUsers := make([]UserJsonOutput, len(users))
	for i := range users {
		outputUsers[i] = UserJsonOutput{
			ID:                &users[i].ID,
			Username:          &users[i].Username,
			FirstName:         &users[i].Profile.FirstName,
			LastName:          &users[i].Profile.LastName,
			Role:              &users[i].Role.Name,
			CreatedTimestamp:  &users[i].Created.Time,
			ModifiedTimestamp: &users[i].Modified.Time,
		}
	}

	if isColumnsChanged {
		filteredMap := make([]map[string]interface{}, len(outputUsers))
		for i := range outputUsers {
			filteredMap[i] = make(map[string]interface{})
			data, _ := json.Marshal(outputUsers[i])
			var resourceMap map[string]interface{}
			json.Unmarshal(data, &resourceMap)

			for _, col := range columns {
				col = strings.ToLower(col)

				if val, ok := resourceMap[col]; ok {
					filteredMap[i][col] = val
				}
			}
		}

		jsonResources, err := json.MarshalIndent(filteredMap, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonResources))
		return nil
	}

	jsonUsers, err := json.MarshalIndent(outputUsers, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonUsers))
	return nil
}

func printTableUsers(columns []string, users []api.User) error {
	data := pterm.TableData{columns}

	for _, user := range users {
		entry := make([]string, len(columns))
		for i := range columns {
			switch strings.ToLower(columns[i]) {
			case "id":
				entry[i] = user.ID
			case "username":
				entry[i] = shellescape.StripUnsafe(user.Username)
			case "firstname":
				entry[i] = shellescape.StripUnsafe(user.Profile.FirstName)
			case "lastname":
				entry[i] = shellescape.StripUnsafe(user.Profile.LastName)
			case "role":
				entry[i] = shellescape.StripUnsafe(user.Role.Name)
			case "createdtimestamp":
				entry[i] = user.Created.Format(time.RFC3339)
			case "modifiedtimestamp":
				entry[i] = user.Modified.Format(time.RFC3339)
			default:
				return fmt.Errorf("Unknown Column: %v", columns[i])
			}
		}
		data = append(data, entry)
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	return nil
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
		return nil, fmt.Errorf("You need to specify atleast one column to return")
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
