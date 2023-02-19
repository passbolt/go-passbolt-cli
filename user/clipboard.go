package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.design/x/clipboard"
)

var UserToClipboardCommand = &cobra.Command{
	Use:     "user",
	Short:   "Copies user column entries to the clipboard.",
	Aliases: []string{"users"},
	RunE:    UserToClipboard,
}

func init() {
	UserToClipboardCommand.Flags().StringP("search", "s", "", "Search for Users")
	UserToClipboardCommand.Flags().StringArrayP("group", "g", []string{}, "Users that are members of groups")
	UserToClipboardCommand.Flags().StringArrayP("column", "c", []string{"Username"}, "Columns to return, possible Columns:\nID, Username, FirstName, LastName, Role, CreatedTimestamp, ModifiedTimestamp")
}

func UserToClipboard(cmd *cobra.Command, args []string) error {
	groups, err := cmd.Flags().GetStringArray("group")
	if err != nil {
		return err
	}
	search, err := cmd.Flags().GetString("search")
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
	celFilter, err := cmd.Flags().GetString("filter")
	if err != nil {
		return err
	}
	delay, err := cmd.Flags().GetInt("delay")
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
		FilterHasGroup: groups,
		FilterSearch:   search,
	})
	if err != nil {
		return fmt.Errorf("Listing User: %w", err)
	}

	users, err = filterUsers(&users, celFilter, ctx)
	if err != nil {
		return err
	}

	singleUser, err := chooseUserEntry(&users)
	if err != nil {
		return err
	}
	if singleUser == nil {
		return nil
	}

	err = clipboard.Init()
	if err != nil {
		return err
	}
	util.SetDelay(delay)

	for _, c := range columns {
		switch strings.ToLower(c) {
		case "id":
			util.CopyValueToClipboard(c, singleUser.ID)
		case "username":
			util.CopyValueToClipboard(c, singleUser.Username)
		case "firstname":
			util.CopyValueToClipboard(c, singleUser.Profile.FirstName)
		case "lastname":
			util.CopyValueToClipboard(c, singleUser.Profile.LastName)
		case "role":
			util.CopyValueToClipboard(c, singleUser.Role.Name)
		case "createdtimestamp":
			util.CopyValueToClipboard(c, singleUser.Created.Format(time.RFC3339))
		case "modifiedtimestamp":
			util.CopyValueToClipboard(c, singleUser.Modified.Format(time.RFC3339))
		default:
			cmd.SilenceUsage = false
			return fmt.Errorf("Unknown Column: %v", c)
		}
	}
	return nil
}

// If more than one user is selected, print an table an ask for selection of entry
func chooseUserEntry(users *[]api.User) (*api.User, error) {
	if len(*users) == 0 {
		return nil, fmt.Errorf("No user to select!")
	}

	if len(*users) == 1 {
		return &(*users)[0], nil
	}

	data := pterm.TableData{[]string{"Index", "Username", "FirstName", "LastName", "Role"}}
	for i, user := range *users {
		index := i + 1
		data = append(data, []string{strconv.Itoa(index), user.Username, user.Profile.FirstName, user.Profile.LastName, user.Role.Name})
	}

	var selectedUser *api.User
	for selectedUser == nil {
		pterm.DefaultTable.WithHasHeader().WithData(data).Render()
		fmt.Print("Please chose an index of user (c to abbort): ")
		var cliInput string
		fmt.Scanln(&cliInput)
		if cliInput == "c" {
			return nil, nil
		}

		selectedIndex, err := strconv.Atoi(cliInput)
		if err != nil || selectedIndex <= 0 || selectedIndex > len(*users) {
			fmt.Printf("Input %s is not a valid index!\n", cliInput)
			fmt.Println()
			continue
		}

		selectedUser = &(*users)[selectedIndex-1]
	}

	return selectedUser, nil
}
