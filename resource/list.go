package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/speatzle/go-passbolt-cli/util"
	"github.com/speatzle/go-passbolt/api"
	"github.com/speatzle/go-passbolt/helper"
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

	ResourceListCmd.Flags().StringArrayP("group", "g", []string{}, "Resources that are shared with group")
	ResourceListCmd.Flags().StringArrayP("folder", "f", []string{}, "Resources that are in folder")

	ResourceListCmd.Flags().StringArrayP("column", "c", []string{"ID", "FolderParentID", "Name", "Username", "URI"}, "Columns to return, possible Columns:\nID, FolderParentID, Name, Username, URI, Password, Description")
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
	groups, err := cmd.Flags().GetStringArray("group")
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
		FilterIsSharedWithGroup: groups,
		FilterHasParent:         folderParents,
	})
	if err != nil {
		return fmt.Errorf("Listing Resource: %w", err)
	}

	data := pterm.TableData{columns}

	for _, resource := range resources {
		entry := make([]string, len(columns))
		for i := range columns {
			switch strings.ToLower(columns[i]) {
			case "id":
				entry[i] = resource.ID
			case "folderparentid":
				entry[i] = resource.FolderParentID
			case "name":
				entry[i] = shellescape.StripUnsafe(resource.Name)
			case "username":
				entry[i] = shellescape.StripUnsafe(resource.Username)
			case "uri":
				entry[i] = shellescape.StripUnsafe(resource.URI)
			case "password":
				_, _, _, _, pass, _, err := helper.GetResource(ctx, client, resource.ID)
				if err != nil {
					return fmt.Errorf("Get Resource %w", err)
				}
				entry[i] = shellescape.StripUnsafe(pass)
			case "description":
				_, _, _, _, _, desc, err := helper.GetResource(ctx, client, resource.ID)
				if err != nil {
					return fmt.Errorf("Get Resource %w", err)
				}
				entry[i] = shellescape.StripUnsafe(desc)
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
