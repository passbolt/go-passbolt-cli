package user

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"

	"github.com/pterm/pterm"
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
	UserListCmd.Flags().StringArrayP("group", "g", []string{}, "Users that are members of groups")
	UserListCmd.Flags().StringArrayP("resource", "r", []string{}, "Users that have access to resources")

	UserListCmd.Flags().StringP("search", "s", "", "Search for Users")
	UserListCmd.Flags().BoolP("admin", "a", false, "Only show Admins")

	UserListCmd.Flags().StringArrayP("column", "c", []string{"ID", "Username", "FirstName", "LastName", "Role"}, "Columns to return, possible Columns:\nID, Username, FirstName, LastName, Role, CreatedTimestamp, ModifiedTimestamp")
}

func UserList(cmd *cobra.Command, args []string) error {
	groups, err := cmd.Flags().GetStringArray("group")
	if err != nil {
		return err
	}
	resources, err := cmd.Flags().GetStringArray("resource")
	if err != nil {
		return err
	}
	search, err := cmd.Flags().GetString("search")
	if err != nil {
		return err
	}
	admin, err := cmd.Flags().GetBool("admin")
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
	celFilter, err := cmd.Flags().GetString("filter")
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

	users, err := client.GetUsers(ctx, &api.GetUsersOptions{
		FilterHasGroup:  groups,
		FilterHasAccess: resources,
		FilterSearch:    search,
		FilterIsAdmin:   admin,
	})
	if err != nil {
		return fmt.Errorf("Listing User: %w", err)
	}

	users, err = filterUsers(&users, celFilter, ctx)
	if err != nil {
		return err
	}

	if jsonOutput {
		outputUsers := []UserJsonOutput{}
		for i := range users {
			outputUsers = append(outputUsers, UserJsonOutput{
				ID:                &users[i].ID,
				Username:          &users[i].Username,
				FirstName:         &users[i].Profile.FirstName,
				LastName:          &users[i].Profile.LastName,
				Role:              &users[i].Role.Name,
				CreatedTimestamp:  &users[i].Created.Time,
				ModifiedTimestamp: &users[i].Modified.Time,
			})
		}
		jsonUsers, err := json.MarshalIndent(outputUsers, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonUsers))
	} else {
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
