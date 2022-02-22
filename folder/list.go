package folder

import (
	"context"
	"fmt"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"

	"github.com/pterm/pterm"
)

// FolderListCmd Lists a Passbolt Folder
var FolderListCmd = &cobra.Command{
	Use:     "folder",
	Short:   "Lists Passbolt Folders",
	Long:    `Lists Passbolt Folders`,
	Aliases: []string{"folders"},
	RunE:    FolderList,
}

func init() {
	FolderListCmd.Flags().StringP("search", "s", "", "Folders that have this in the Name")
	FolderListCmd.Flags().StringArrayP("folder", "f", []string{}, "Folders that are in this Folder")
	FolderListCmd.Flags().StringArrayP("group", "g", []string{}, "Folders that are shared with group")
	FolderListCmd.Flags().StringArrayP("column", "c", []string{"ID", "FolderParentID", "Name"}, "Columns to return, possible Columns:\nID, FolderParentID, Name")
}

func FolderList(cmd *cobra.Command, args []string) error {
	search, err := cmd.Flags().GetString("search")
	if err != nil {
		return err
	}
	parentFolders, err := cmd.Flags().GetStringArray("folder")
	if err != nil {
		return err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return fmt.Errorf("You need to Specify atleast one column to return")
	}

	ctx := util.GetContext()
	cmd.SilenceUsage = true

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())

	folders, err := client.GetFolders(ctx, &api.GetFoldersOptions{
		FilterHasParent: parentFolders,
		FilterSearch:    search,
	})
	if err != nil {
		return fmt.Errorf("Listing Folder: %w", err)
	}

	data := pterm.TableData{columns}

	for _, folder := range folders {
		entry := make([]string, len(columns))
		for i := range columns {
			switch strings.ToLower(columns[i]) {
			case "id":
				entry[i] = folder.ID
			case "folderparentid":
				entry[i] = folder.FolderParentID
			case "name":
				entry[i] = shellescape.StripUnsafe(folder.Name)
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
