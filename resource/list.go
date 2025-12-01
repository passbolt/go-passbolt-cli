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

// ResourceListCmd Lists a Passbolt Resource
var ResourceListCmd = &cobra.Command{
	Use:     "resource",
	Short:   "Lists Passbolt Resources",
	Long:    `Lists Passbolt Resources`,
	Aliases: []string{"resources"},
	RunE:    ResourceList,
}

func init() {
	ResourceListCmd.Flags().Bool("favorite", false, "Resources that are marked as favorite")
	ResourceListCmd.Flags().Bool("own", false, "Resources that are owned by me")
	ResourceListCmd.Flags().StringP("group", "g", "", "Resources that are shared with group")
	ResourceListCmd.Flags().StringArrayP("folder", "f", []string{}, "Resources that are in folder")
	ResourceListCmd.Flags().StringArrayP("column", "c", []string{"ID", "FolderParentID", "Name", "Username", "URI"}, "Columns to return, possible Columns:\nID, FolderParentID, Name, Username, URI, Password, Description, CreatedTimestamp, ModifiedTimestamp")
}

func ResourceList(cmd *cobra.Command, args []string) error {
	favorite, err := cmd.Flags().GetBool("favorite")
	if err != nil {
		return err
	}
	own, err := cmd.Flags().GetBool("own")
	if err != nil {
		return err
	}
	group, err := cmd.Flags().GetString("group")
	if err != nil {
		return err
	}
	folderParents, err := cmd.Flags().GetStringArray("folder")
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

	resources, err := client.GetResources(ctx, &api.GetResourcesOptions{
		FilterIsFavorite:        favorite,
		FilterIsOwnedByMe:       own,
		FilterIsSharedWithGroup: group,
		FilterHasParent:         folderParents,
	})
	if err != nil {
		return fmt.Errorf("Listing Resource: %w", err)
	}

	resources, err = filterResources(&resources, celFilter, ctx, client)
	if err != nil {
		return err
	}

	if jsonOutput {
		outputResources := make([]ResourceJsonOutput, len(resources))
		for i := range resources {
			_, name, username, uri, pass, desc, err := helper.GetResource(ctx, client, resources[i].ID)
			if err != nil {
				return fmt.Errorf("Get Resource %w", err)
			}

			fullResource := ResourceJsonOutput{
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
			outputResources[i] = fullResource
		}

		if cmd.Flags().Changed("column") && len(columns) > 0 {
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
	} else {
		data := pterm.TableData{columns}

		for _, resource := range resources {
			// TODO We should decrypt the secret only when required for performance reasonse
			_, name, username, uri, pass, desc, err := helper.GetResource(ctx, client, resource.ID)
			if err != nil {
				return fmt.Errorf("Get Resource %w", err)
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
