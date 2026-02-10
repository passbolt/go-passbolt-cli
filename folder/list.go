package folder

import (
	"context"
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

var defaultTableColumns = []string{"ID", "FolderParentID", "Name"}

// FolderListCmd Lists a Passbolt Folder
var FolderListCmd = &cobra.Command{
	Use:     "folder",
	Short:   "Lists Passbolt Folders",
	Long:    `Lists Passbolt Folders`,
	Aliases: []string{"folders"},
	RunE:    FolderList,
}

func init() {
	flags := FolderListCmd.Flags()
	flags.StringP("search", "s", "", "Folders that have this in the Name")
	flags.StringArrayP("folder", "f", []string{}, "Folders that are in this Folder")
	flags.StringArrayP("group", "g", []string{}, "Folders that are shared with group")
	flags.StringArrayP("column", "c", defaultTableColumns, "Columns to return (default list only for table format; JSON format includes all fields by default).\nPossible Columns: ID, FolderParentID, Name, CreatedTimestamp, ModifiedTimestamp")
}

type folderListConfig struct {
	search         string
	parentFolders  []string
	columns        []string
	columnsChanged bool
	jsonOutput     bool
	celFilter      string
}

func FolderList(cmd *cobra.Command, args []string) error {
	config, err := parseFolderListFlags(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	folders, err := client.GetFolders(ctx, &api.GetFoldersOptions{
		FilterHasParent: config.parentFolders,
		FilterSearch:    config.search,
	})
	if err != nil {
		return fmt.Errorf("Listing Folder: %w", err)
	}

	folders, err = filterFolders(&folders, config.celFilter, ctx)
	if err != nil {
		return err
	}

	if config.jsonOutput {
		return printJsonFolders(folders, config.columnsChanged, config.columns)
	}

	return printTableFolders(config.columns, folders)
}

func printJsonFolders(folders []api.Folder, isColumnsChanged bool, columns []string) error {
	outputFolders := make([]FolderJsonOutput, len(folders))
	for i := range folders {
		outputFolders[i] = FolderJsonOutput{
			ID:                &folders[i].ID,
			FolderParentID:    &folders[i].FolderParentID,
			Name:              &folders[i].Name,
			CreatedTimestamp:  &folders[i].Created.Time,
			ModifiedTimestamp: &folders[i].Modified.Time,
		}
	}

	if isColumnsChanged {
		filteredMap := make([]map[string]interface{}, len(outputFolders))
		for i := range outputFolders {
			filteredMap[i] = make(map[string]interface{})
			data, _ := json.Marshal(outputFolders[i])
			var folderMap map[string]interface{}
			json.Unmarshal(data, &folderMap)

			for _, col := range columns {
				col = strings.ToLower(col)

				if val, ok := folderMap[col]; ok {
					filteredMap[i][col] = val
				}
			}
		}

		jsonFolders, err := json.MarshalIndent(filteredMap, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonFolders))
		return nil
	}

	jsonFolders, err := json.MarshalIndent(outputFolders, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonFolders))
	return nil
}

func printTableFolders(columns []string, folders []api.Folder) error {
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
			case "createdtimestamp":
				entry[i] = folder.Created.Format(time.RFC3339)
			case "modifiedtimestamp":
				entry[i] = folder.Modified.Format(time.RFC3339)
			default:
				return fmt.Errorf("Unknown Column: %v", columns[i])
			}
		}
		data = append(data, entry)
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	return nil
}

func parseFolderListFlags(cmd *cobra.Command) (*folderListConfig, error) {
	search, err := cmd.Flags().GetString("search")
	if err != nil {
		return nil, err
	}
	parentFolders, err := cmd.Flags().GetStringArray("folder")
	if err != nil {
		return nil, err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("You need to specify at least one column to return")
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return nil, err
	}
	celFilter, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return &folderListConfig{
		search:         search,
		parentFolders:  parentFolders,
		columns:        columns,
		columnsChanged: cmd.Flags().Changed("column"),
		jsonOutput:     jsonOutput,
		celFilter:      celFilter,
	}, nil
}
