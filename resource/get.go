package resource

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"al.essio.dev/pkg/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// ResourceGetCmd Gets a Passbolt Resource
var ResourceGetCmd = &cobra.Command{
	Use:   "resource",
	Short: "Gets a Passbolt Resource",
	Long:  `Gets a Passbolt Resource`,
	RunE:  ResourceGet,
}

// ResourcePermissionCmd Gets Permissions for Passbolt Resource
var ResourcePermissionCmd = &cobra.Command{
	Use:     "permission",
	Short:   "Gets Permissions for a Passbolt Resource",
	Long:    `Gets Permissions for a Passbolt Resource`,
	Aliases: []string{"permissions"},
	RunE:    ResourcePermission,
}

func init() {
	ResourceGetCmd.Flags().String("id", "", "id of Resource to Get")

	ResourceGetCmd.MarkFlagRequired("id")

	ResourceGetCmd.AddCommand(ResourcePermissionCmd)
	ResourcePermissionCmd.Flags().String("id", "", "id of Resource to Get")
	ResourcePermissionCmd.Flags().StringArrayP("column", "c", []string{"ID", "Aco", "AcoForeignKey", "Aro", "AroForeignKey", "Type"}, "Columns to return, possible Columns:\nID, Aco, AcoForeignKey, Aro, AroForeignKey, Type, CreatedTimestamp, ModifiedTimestamp")

	ResourcePermissionCmd.MarkFlagRequired("id")
}

func ResourceGet(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	// Use the Resource wrapper for dynamic field access
	r, err := helper.FetchResourceWithSecret(ctx, client, id)
	if err != nil {
		return fmt.Errorf("getting resource: %w", err)
	}

	folderParentID := r.FolderParentID()
	name, err := r.Name(ctx)
	if err != nil {
		return fmt.Errorf("decrypting resource name: %w", err)
	}
	username, err := r.Username(ctx)
	if err != nil {
		return fmt.Errorf("decrypting resource username: %w", err)
	}
	uri, err := r.URI(ctx)
	if err != nil {
		return fmt.Errorf("decrypting resource uri: %w", err)
	}
	password, err := r.Password(ctx)
	if err != nil {
		return fmt.Errorf("decrypting resource password: %w", err)
	}
	description, err := r.Description(ctx)
	if err != nil {
		return fmt.Errorf("decrypting resource description: %w", err)
	}

	if jsonOutput {
		output := ResourceJSONOutput{
			FolderParentID: &folderParentID,
			Name:           &name,
			Username:       &username,
			URI:            &uri,
			Password:       &password,
			Description:    &description,
		}

		// Include full metadata and secret maps for richer output
		metadata, err := r.MetadataFields(ctx)
		if err != nil {
			return fmt.Errorf("getting metadata fields: %w", err)
		}
		secretFields, err := r.SecretFields(ctx)
		if err != nil {
			return fmt.Errorf("getting secret fields: %w", err)
		}
		if len(metadata) > 0 {
			output.Metadata = metadata
		}
		if len(secretFields) > 0 {
			output.Secret = secretFields
		}

		jsonResource, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonResource))
	} else {
		fmt.Printf("FolderParentID: %v\n", folderParentID)
		fmt.Printf("Name: %v\n", shellescape.StripUnsafe(name))
		fmt.Printf("Username: %v\n", shellescape.StripUnsafe(username))
		fmt.Printf("URI: %v\n", shellescape.StripUnsafe(uri))
		fmt.Printf("Password: %v\n", shellescape.StripUnsafe(password))
		fmt.Printf("Description: %v\n", shellescape.StripUnsafe(description))

		// Show additional metadata fields not covered by standard output
		metadata, err := r.MetadataFields(ctx)
		if err != nil {
			return fmt.Errorf("getting metadata fields: %w", err)
		}
		for k, v := range metadata {
			switch k {
			case "name", "username", "uri", "uris", "description", "object_type", "resource_type_id":
				continue
			default:
				fmt.Printf("%s: %v\n", k, shellescape.StripUnsafe(fmt.Sprint(v)))
			}
		}
	}
	return nil
}

func ResourcePermission(cmd *cobra.Command, args []string) error {
	resource, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return fmt.Errorf("you need to specify at least one column to return")
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	permissions, err := client.GetResourcePermissions(ctx, resource)
	if err != nil {
		return fmt.Errorf("listing Permission: %w", err)
	}

	if jsonOutput {
		outputPermissions := []util.PermissionJSONOutput{}
		for i := range permissions {
			outputPermissions = append(outputPermissions, util.PermissionJSONOutput{
				ID:                &permissions[i].ID,
				Aco:               &permissions[i].ACO,
				AcoForeignKey:     &permissions[i].ACOForeignKey,
				Aro:               &permissions[i].ARO,
				AroForeignKey:     &permissions[i].AROForeignKey,
				Type:              &permissions[i].Type,
				CreatedTimestamp:  &permissions[i].Created.Time,
				ModifiedTimestamp: &permissions[i].Modified.Time,
			})
		}
		jsonPermissions, err := json.MarshalIndent(outputPermissions, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonPermissions))
	} else {
		data := pterm.TableData{columns}

		for _, permission := range permissions {
			entry := make([]string, len(columns))
			for i := range columns {
				switch strings.ToLower(columns[i]) {
				case "id":
					entry[i] = permission.ID
				case "aco":
					entry[i] = permission.ACO
				case "acoforeignkey":
					entry[i] = permission.ACOForeignKey
				case "aro":
					entry[i] = permission.ARO
				case "aroforeignkey":
					entry[i] = permission.AROForeignKey
				case "type":
					entry[i] = strconv.Itoa(permission.Type)
				case "createdtimestamp":
					entry[i] = permission.Created.Format(time.RFC3339)
				case "modifiedtimestamp":
					entry[i] = permission.Modified.Format(time.RFC3339)
				default:
					cmd.SilenceUsage = false
					return fmt.Errorf("unknown Column: %v", columns[i])
				}
			}
			data = append(data, entry)
		}

		pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	}

	return nil
}
