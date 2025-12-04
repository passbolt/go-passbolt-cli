package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"al.essio.dev/pkg/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"

	"github.com/pterm/pterm"
)

var defaultTableColumns = []string{"ID", "FolderParentID", "Name", "Username", "URI"}

// ResourceListCmd Lists a Passbolt Resource
var ResourceListCmd = &cobra.Command{
	Use:     "resource",
	Short:   "Lists Passbolt Resources",
	Long:    `Lists Passbolt Resources`,
	Aliases: []string{"resources"},
	RunE:    ResourceList,
}

func init() {
	flags := ResourceListCmd.Flags()
	flags.Bool("favorite", false, "Resources that are marked as favorite")
	flags.Bool("own", false, "Resources that are owned by me")
	flags.StringP("group", "g", "", "Resources that are shared with group")
	flags.StringArrayP("folder", "f", []string{}, "Resources that are in folder")
	flags.StringArrayP("column", "c", defaultTableColumns, "Columns to return (default list only for table format; JSON format includes all fields by default).\nPossible Columns: ID, FolderParentID, Name, Username, URI, Password, Description, CreatedTimestamp, ModifiedTimestamp")
}

type resourceListConfig struct {
	favorite       bool
	own            bool
	group          string
	folderParents  []string
	columns        []string
	columnsChanged bool
	jsonOutput     bool
	celFilter      string
}

func ResourceList(cmd *cobra.Command, args []string) error {
	config, err := parseResourceListFlags(cmd)
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

	resources, err := client.GetResources(ctx, &api.GetResourcesOptions{
		FilterIsFavorite:        config.favorite,
		FilterIsOwnedByMe:       config.own,
		FilterIsSharedWithGroup: config.group,
		FilterHasParent:         config.folderParents,
	})
	if err != nil {
		return fmt.Errorf("listing resource: %w", err)
	}

	resources, err = filterResources(&resources, config.celFilter, ctx, client)
	if err != nil {
		return err
	}

	if config.jsonOutput {
		return printJsonResources(ctx, client, resources, config.columnsChanged, config.columns)
	}

	return printTableResources(ctx, client, resources, config.columns)
}

func printJsonResources(
	ctx context.Context,
	client *api.Client,
	resources []api.Resource,
	isColumnsChanged bool,
	columns []string,
) error {
	outputResources := make([]ResourceJsonOutput, len(resources))
	for i := range resources {
		_, name, username, uri, pass, desc, err := helper.GetResource(ctx, client, resources[i].ID)
		if err != nil {
			return fmt.Errorf("get resource %w", err)
		}

		outputResources[i] = ResourceJsonOutput{
			ID:                &resources[i].ID,
			FolderParentID:    &resources[i].FolderParentID,
			Name:              &name,
			Username:          &username,
			URI:               &uri,
			Password:          &pass,
			Description:       &desc,
			CreatedTimestamp:  &resources[i].Created.Time,
			ModifiedTimestamp: &resources[i].Modified.Time,
		}
	}

	if isColumnsChanged {
		filteredMap := make([]map[string]interface{}, len(outputResources))
		for i := range outputResources {
			filteredMap[i] = make(map[string]interface{})
			data, _ := json.Marshal(outputResources[i])
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

	jsonResources, err := json.MarshalIndent(outputResources, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonResources))
	return nil
}

func printTableResources(
	ctx context.Context,
	client *api.Client,
	resources []api.Resource,
	columns []string,
) error {
	data := pterm.TableData{columns}

	for _, resource := range resources {
		// TODO We should decrypt the secret only when required for performance reasons
		_, name, username, uri, pass, desc, err := helper.GetResource(ctx, client, resource.ID)
		if err != nil {
			return fmt.Errorf("get resource %w", err)
		}

		entry := make([]string, len(columns))
		for i := range columns {
			switch strings.ToLower(columns[i]) {
			case "id":
				entry[i] = resource.ID
			case "folderparentid":
				entry[i] = resource.FolderParentID
			case "name":
				entry[i] = shellescape.StripUnsafe(name)
			case "username":
				entry[i] = shellescape.StripUnsafe(username)
			case "uri":
				entry[i] = shellescape.StripUnsafe(uri)
			case "password":
				entry[i] = shellescape.StripUnsafe(pass)
			case "description":
				entry[i] = shellescape.StripUnsafe(desc)
			case "createdtimestamp":
				entry[i] = resource.Created.Format(time.RFC3339)
			case "modifiedtimestamp":
				entry[i] = resource.Modified.Format(time.RFC3339)
			default:
				return fmt.Errorf("unknown column: %v", columns[i])
			}
		}
		data = append(data, entry)
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	return nil
}

func parseResourceListFlags(cmd *cobra.Command) (*resourceListConfig, error) {
	favorite, err := cmd.Flags().GetBool("favorite")
	if err != nil {
		return nil, err
	}
	own, err := cmd.Flags().GetBool("own")
	if err != nil {
		return nil, err
	}
	group, err := cmd.Flags().GetString("group")
	if err != nil {
		return nil, err
	}
	folderParents, err := cmd.Flags().GetStringArray("folder")
	if err != nil {
		return nil, err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("you need to specify at least one column to return")
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return nil, err
	}
	celFilter, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return &resourceListConfig{
		favorite:       favorite,
		own:            own,
		group:          group,
		folderParents:  folderParents,
		columns:        columns,
		columnsChanged: cmd.Flags().Changed("column"),
		jsonOutput:     jsonOutput,
		celFilter:      celFilter,
	}, nil
}
