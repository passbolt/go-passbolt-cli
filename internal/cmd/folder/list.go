package folder

import (
	"context"
	"fmt"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

// FolderListCmd Lists a Passbolt Folder
var FolderListCmd = makeFolderListCmd()

func makeFolderListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "folder",
		Short:   "Lists Passbolt Folders",
		Long:    `Lists Passbolt Folders`,
		Aliases: []string{"folders"},
		RunE:    FolderList,
	}
}

func configureList() {
	flags := FolderListCmd.Flags()
	flags.StringP("search", "s", "", "Folders that have this in the Name")
	flags.StringArrayP("folder", "f", []string{}, "Folders that are in this Folder")
	flags.StringArrayP("group", "g", []string{}, "Folders that are shared with group")
	flags.StringArrayP("column", "c", folderColumns.DefaultTableColumns(), "Columns to return (default list only for table format; JSON format includes all fields by default).\nPossible Columns: "+strings.Join(folderColumns.Resolver().Canonical(), ", ")+"\nLegacy PascalCase column names (ID, FolderParentID, ...) remain accepted for backwards compatibility.")
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

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		folders, err := client.GetFolders(ctx, &api.GetFoldersOptions{
			FilterHasParent: config.parentFolders,
			FilterSearch:    config.search,
		})
		if err != nil {
			return fmt.Errorf("listing Folder: %w", err)
		}

		folders, err = folderColumns.Filter(ctx, folders, config.celFilter)
		if err != nil {
			return err
		}

		if config.jsonOutput {
			return printJSONFolders(folders, config.columnsChanged, config.columns)
		}

		return printTableFolders(config.columns, folders)
	})
}

func printJSONFolders(folders []api.Folder, isColumnsChanged bool, columns []string) error {
	outputFolders := make([]FolderJSONOutput, len(folders))
	for i := range folders {
		outputFolders[i] = FolderJSONOutput{
			ID:                &folders[i].ID,
			FolderParentID:    &folders[i].FolderParentID,
			Name:              &folders[i].Name,
			CreatedTimestamp:  &folders[i].Created.Time,
			ModifiedTimestamp: &folders[i].Modified.Time,
			Personal:          folders[i].Personal,
		}
	}

	if isColumnsChanged {
		return util.PrintJSONColumnFiltered(outputFolders, columns)
	}

	return util.PrintJSON(outputFolders)
}

func printTableFolders(columns []string, folders []api.Folder) error {
	// Input is normalized by parseFolderListFlags; a miss in the resolver is a
	// defensive guard against a future caller that skips that step.
	return util.PrintTable(columns, folders, folderColumns.TableValue)
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
		return nil, util.NoColumnsError(folderColumns.Resolver().Canonical())
	}
	columns, err = folderColumns.Resolver().NormalizeAll(columns)
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

	return &folderListConfig{
		search:         search,
		parentFolders:  parentFolders,
		columns:        columns,
		columnsChanged: cmd.Flags().Changed("column"),
		jsonOutput:     jsonOutput,
		celFilter:      celFilter,
	}, nil
}

func init() { configureList() }
