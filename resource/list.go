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

	// Check if we need to fetch secrets (password column or any encrypted field when not available)
	// For Passbolt v5+, we should fetch secrets to get all encrypted fields in one request
	needsSecrets := false
	for _, col := range columns {
		colLower := strings.ToLower(col)
		if colLower == "password" || colLower == "name" || colLower == "username" ||
		   colLower == "uri" || colLower == "description" {
			needsSecrets = true
			break
		}
	}

	// Also check if filter uses encrypted fields
	if !needsSecrets && celFilter != "" {
		encryptedFields := []string{"Name", "Username", "URI", "Password", "Description"}
		for _, field := range encryptedFields {
			if strings.Contains(celFilter, field) {
				needsSecrets = true
				break
			}
		}
	}

	resources, err := client.GetResources(ctx, &api.GetResourcesOptions{
		FilterIsFavorite:        favorite,
		FilterIsOwnedByMe:       own,
		FilterIsSharedWithGroup: group,
		FilterHasParent:         folderParents,
		ContainSecret:           needsSecrets,
	})
	if err != nil {
		return fmt.Errorf("Listing Resource: %w", err)
	}

	resources, err = filterResources(&resources, celFilter, ctx, client)
	if err != nil {
		return err
	}

	if jsonOutput {
		// Cache resource types to avoid fetching the same type repeatedly
		resourceTypeCache := make(map[string]*api.ResourceType)

		outputResources := []map[string]interface{}{}
		for i := range resources {
			decrypted, err := decryptResource(ctx, client, resources[i], needsSecrets, resourceTypeCache)
			if err != nil {
				return err
			}

			// Build output with only requested columns
			output := make(map[string]interface{})
			for _, col := range columns {
				switch strings.ToLower(col) {
				case "id":
					output["ID"] = resources[i].ID
				case "folderparentid":
					output["FolderParentID"] = resources[i].FolderParentID
				case "name":
					output["Name"] = decrypted.name
				case "username":
					output["Username"] = decrypted.username
				case "uri":
					output["URI"] = decrypted.uri
				case "password":
					output["Password"] = decrypted.password
				case "description":
					output["Description"] = decrypted.description
				case "createdtimestamp":
					output["CreatedTimestamp"] = resources[i].Created.Time
				case "modifiedtimestamp":
					output["ModifiedTimestamp"] = resources[i].Modified.Time
				}
			}

			outputResources = append(outputResources, output)
		}
		jsonResources, err := json.MarshalIndent(outputResources, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonResources))
	} else {
		data := pterm.TableData{columns}

		// Check if we need to fetch encrypted secrets (Password always requires decryption)
		needsPassword := false
		for _, col := range columns {
			if strings.ToLower(col) == "password" {
				needsPassword = true
				break
			}
		}

		// Cache resource types to avoid fetching the same type repeatedly
		resourceTypeCache := make(map[string]*api.ResourceType)

		for _, resource := range resources {
			var err error

			// Decrypt resource if needed
			decrypted, err := decryptResource(ctx, client, resource, needsSecrets, resourceTypeCache)
			if err != nil {
				return err
			}

			// Fallback: If we need password but secrets weren't included, fetch individually
			if needsPassword && len(resource.Secrets) == 0 {
				_, decrypted.name, decrypted.username, decrypted.uri, decrypted.password, decrypted.description, err = helper.GetResource(ctx, client, resource.ID)
				if err != nil {
					return fmt.Errorf("Get Resource %w", err)
				}
			}

			entry := make([]string, len(columns))
			for i := range columns {
				switch strings.ToLower(columns[i]) {
				case "id":
					entry[i] = resource.ID
				case "folderparentid":
					entry[i] = resource.FolderParentID
				case "name":
					entry[i] = shellescape.StripUnsafe(decrypted.name)
				case "username":
					entry[i] = shellescape.StripUnsafe(decrypted.username)
				case "uri":
					entry[i] = shellescape.StripUnsafe(decrypted.uri)
				case "password":
					entry[i] = shellescape.StripUnsafe(decrypted.password)
				case "description":
					entry[i] = shellescape.StripUnsafe(decrypted.description)
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
