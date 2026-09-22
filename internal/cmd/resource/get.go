package resource

import (
	"context"
	"fmt"

	"al.essio.dev/pkg/shellescape"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
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

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		resource, err := client.GetResource(ctx, id)
		if err != nil {
			return fmt.Errorf("getting resource: %w", err)
		}
		rType, err := client.GetResourceType(ctx, resource.ResourceTypeID)
		if err != nil {
			return fmt.Errorf("getting resource type: %w", err)
		}
		secret, err := client.GetSecret(ctx, resource.ID)
		if err != nil {
			return fmt.Errorf("getting secret: %w", err)
		}

		folderParentID, metadata, secretFields, err :=
			helper.GetResourceFieldMaps(client, *resource, *secret, *rType, true)
		if err != nil {
			return util.ExplainReadError("decrypting Resource", rType.Slug, err)
		}

		name := helper.GetStringField(metadata, "name")
		username := helper.GetStringField(metadata, "username")
		uri := helper.GetStringField(metadata, "uri")
		description := helper.GetStringField(metadata, "description")
		password := helper.GetStringField(secretFields, "password")

		if jsonOutput {
			output := ResourceJSONOutput{
				FolderParentID: &folderParentID,
				Name:           &name,
				Username:       &username,
				URI:            &uri,
				Password:       &password,
				Description:    &description,
			}
			if len(metadata) > 0 {
				output.Metadata = metadata
			}
			if len(secretFields) > 0 {
				output.Secret = secretFields
			}

			return util.PrintJSON(output)
		}

		fmt.Printf("FolderParentID: %v\n", folderParentID)
		fmt.Printf("Name: %v\n", shellescape.StripUnsafe(name))
		fmt.Printf("Username: %v\n", shellescape.StripUnsafe(username))
		fmt.Printf("URI: %v\n", shellescape.StripUnsafe(uri))
		fmt.Printf("Password: %v\n", shellescape.StripUnsafe(password))
		fmt.Printf("Description: %v\n", shellescape.StripUnsafe(description))

		for k, v := range metadata {
			switch k {
			case "name", "username", "uri", "uris", "description", "object_type", "resource_type_id":
				continue
			default:
				fmt.Printf("%s: %v\n", k, shellescape.StripUnsafe(fmt.Sprint(v)))
			}
		}
		return nil
	})
}

func ResourcePermission(cmd *cobra.Command, args []string) error {
	resourceID, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return util.NoColumnsError(util.PermissionColumns)
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		permissions, err := client.GetResourcePermissions(ctx, resourceID)
		if err != nil {
			return fmt.Errorf("listing Permission for Resource %s: %w", resourceID, err)
		}

		if jsonOutput {
			return util.PrintJSON(util.PermissionsToJSONOutput(permissions))
		}

		if err := util.PrintPermissionTable(columns, permissions); err != nil {
			cmd.SilenceUsage = false
			return err
		}
		return nil
	})
}
