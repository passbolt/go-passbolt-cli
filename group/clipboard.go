package group

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.design/x/clipboard"
)

var GroupToClipboardCommand = &cobra.Command{
	Use:     "group",
	Short:   "Copies group column entries to the clipboard.",
	Aliases: []string{"groups"},
	RunE:    GroupToClipboard,
}

func init() {
	GroupToClipboardCommand.Flags().StringArrayP("column", "c", []string{"Name"}, "Columns to return, possible Columns:\nID, FolderParentID, Name, CreatedTimestamp, ModifiedTimestamp")
}

func GroupToClipboard(cmd *cobra.Command, args []string) error {
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

	groups, err := client.GetGroups(ctx, &api.GetGroupsOptions{})
	if err != nil {
		return fmt.Errorf("Listing Group: %w", err)
	}

	groups, err = filterGroups(&groups, celFilter, ctx)
	if err != nil {
		return err
	}

	singleGroup, err := chooseGroupEntry(&groups)
	if err != nil {
		return err
	}
	if singleGroup == nil {
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
			util.CopyValueToClipboard(c, singleGroup.ID)
		case "name":
			util.CopyValueToClipboard(c, shellescape.StripUnsafe(singleGroup.Name))
		case "createdtimestamp":
			util.CopyValueToClipboard(c, singleGroup.Created.Format(time.RFC3339))
		case "modifiedtimestamp":
			util.CopyValueToClipboard(c, singleGroup.Modified.Format(time.RFC3339))
		default:
			cmd.SilenceUsage = false
			return fmt.Errorf("Unknown Column: %v", c)
		}
	}
	return nil
}

// If more than one groups are selected, print an table an ask for selection of entry
func chooseGroupEntry(groups *[]api.Group) (*api.Group, error) {
	if len(*groups) == 0 {
		return nil, fmt.Errorf("No group to select!")
	}

	if len(*groups) == 1 {
		return &(*groups)[0], nil
	}

	data := pterm.TableData{[]string{"Index", "Name"}}
	for i, group := range *groups {
		index := i + 1
		data = append(data, []string{strconv.Itoa(index), group.Name})
	}

	var selectedGroup *api.Group
	for selectedGroup == nil {
		pterm.DefaultTable.WithHasHeader().WithData(data).Render()
		fmt.Print("Please chose an index of group (c to abbort): ")
		var cliInput string
		fmt.Scanln(&cliInput)
		if cliInput == "c" {
			return nil, nil
		}

		selectedIndex, err := strconv.Atoi(cliInput)
		if err != nil || selectedIndex <= 0 || selectedIndex > len(*groups) {
			fmt.Printf("Input %s is not a valid index!\n", cliInput)
			fmt.Println()
			continue
		}

		selectedGroup = &(*groups)[selectedIndex-1]
	}

	return selectedGroup, nil
}
