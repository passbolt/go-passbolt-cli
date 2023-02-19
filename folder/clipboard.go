package folder

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

var FolderToClipboardCommand = &cobra.Command{
	Use:     "folder",
	Short:   "Copies folder column entries to the clipboard.",
	Aliases: []string{"folders"},
	RunE:    FolderToClipboard,
}

func init() {
	FolderToClipboardCommand.Flags().StringP("search", "s", "", "Folders that have this in the Name")
	FolderToClipboardCommand.Flags().StringArrayP("column", "c", []string{"Name"}, "Columns to return, possible Columns:\nID, FolderParentID, Name, CreatedTimestamp, ModifiedTimestamp")
}

func FolderToClipboard(cmd *cobra.Command, args []string) error {
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

	folders, err := client.GetFolders(ctx, &api.GetFoldersOptions{
		FilterSearch: search,
	})
	if err != nil {
		return fmt.Errorf("Listing Folder: %w", err)
	}

	folders, err = filterFolders(&folders, celFilter, ctx)
	if err != nil {
		return err
	}

	singleFolder, err := choseFolderEntry(&folders)
	if err != nil {
		return err
	}
	if singleFolder == nil {
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
			util.CopyValueToClipboard(c, singleFolder.ID)
		case "folderparentid":
			util.CopyValueToClipboard(c, singleFolder.FolderParentID)
		case "name":
			util.CopyValueToClipboard(c, shellescape.StripUnsafe(singleFolder.Name))
		case "createdtimestamp":
			util.CopyValueToClipboard(c, singleFolder.Created.Format(time.RFC3339))
		case "modifiedtimestamp":
			util.CopyValueToClipboard(c, singleFolder.Modified.Format(time.RFC3339))
		default:
			cmd.SilenceUsage = false
			return fmt.Errorf("Unknown Column: %v", c)
		}
	}
	return nil
}

// If more than one folders are selected, print an table an ask for selection of entry
func choseFolderEntry(folders *[]api.Folder) (*api.Folder, error) {
	if len(*folders) == 0 {
		return nil, fmt.Errorf("No folders to select!")
	}

	if len(*folders) == 1 {
		return &(*folders)[0], nil
	}

	data := pterm.TableData{[]string{"Index", "Name"}}
	for i, folder := range *folders {
		index := i + 1
		data = append(data, []string{strconv.Itoa(index), folder.Name})
	}

	var selectedFolder *api.Folder
	for selectedFolder == nil {
		pterm.DefaultTable.WithHasHeader().WithData(data).Render()
		fmt.Print("Please chose an index of folder (c to abbort): ")
		var cliInput string
		fmt.Scanln(&cliInput)
		if cliInput == "c" {
			return nil, nil
		}

		selectedIndex, err := strconv.Atoi(cliInput)
		if err != nil || selectedIndex <= 0 || selectedIndex > len(*folders) {
			fmt.Printf("Input %s is not a valid index!\n", cliInput)
			fmt.Println()
			continue
		}

		selectedFolder = &(*folders)[selectedIndex-1]
	}

	return selectedFolder, nil
}
