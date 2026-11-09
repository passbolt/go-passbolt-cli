package resource

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/credential"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// ResourceGetCmd Gets a Passbolt Resource
var ResourceGetCmd = makeResourceGetCmd()

func makeResourceGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resource",
		Short: "Gets a Passbolt Resource",
		Long:  `Gets a Passbolt Resource`,
		RunE:  ResourceGet,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			_, _, err := parseRawFlags(cmd)
			return err
		},
	}
}

// ResourcePermissionCmd Gets Permissions for Passbolt Resource
var ResourcePermissionCmd = makeResourcePermissionCmd()

func makeResourcePermissionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "permission",
		Short:   "Gets Permissions for a Passbolt Resource",
		Long:    `Gets Permissions for a Passbolt Resource`,
		Aliases: []string{"permissions"},
		RunE:    ResourcePermission,
	}
}

func configureGet() {
	ResourceGetCmd.Flags().String("id", "", "id of Resource to Get")
	ResourceGetCmd.Flags().Bool("raw", false, "Print one string field with no formatting or added newline")
	ResourceGetCmd.Flags().String("field", "password", "Field to print with --raw (defaults to password)")

	ResourceGetCmd.MarkFlagRequired("id")

	ResourceGetCmd.AddCommand(ResourcePermissionCmd)
	ResourcePermissionCmd.Flags().String("id", "", "id of Resource to Get")
	ResourcePermissionCmd.Flags().StringArrayP("column", "c", []string{"ID", "Aco", "AcoForeignKey", "Aro", "AroForeignKey", "Type"}, "Columns to return, possible Columns:\nID, Aco, AcoForeignKey, Aro, AroForeignKey, Type, CreatedTimestamp, ModifiedTimestamp")

	ResourcePermissionCmd.MarkFlagRequired("id")
}

func ResourceGet(cmd *cobra.Command, args []string) error {
	raw, field, err := parseRawFlags(cmd)
	if err != nil {
		return err
	}
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	runner := util.WithClient
	if raw {
		runner = util.WithMachineClient
	}
	runErr := runner(cmd, func(ctx context.Context, client *api.Client) error {
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

		if raw {
			value, err := extractRawField(metadata, secretFields, field)
			if err != nil {
				return err
			}
			_, err = io.WriteString(util.DataOut, strings.TrimSpace(value))
			return err
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

		util.Printf("FolderParentID: %v\n", util.Sanitize(folderParentID))
		util.Printf("Name: %v\n", util.Sanitize(name))
		util.Printf("Username: %v\n", util.Sanitize(username))
		util.Printf("URI: %v\n", util.Sanitize(uri))
		util.Printf("Password: %v\n", util.Sanitize(password))
		// No Sanitize: a description may span several lines, and the writer cleans it.
		util.Printf("Description: %v\n", description)

		for k, v := range metadata {
			switch k {
			case "name", "username", "uri", "uris", "description", "object_type", "resource_type_id":
				continue
			default:
				util.Printf("%s: %v\n", util.Sanitize(k), util.Sanitize(fmt.Sprint(v)))
			}
		}
		return nil
	})
	if raw && runErr != nil {
		return fmt.Errorf("raw resource field retrieval failed; check the resource ID, field and authentication")
	}
	return runErr

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

func init() { configureGet() }

func parseRawFlags(cmd *cobra.Command) (bool, string, error) {
	raw, err := cmd.Flags().GetBool("raw")
	if err != nil {
		return false, "", err
	}
	field, err := cmd.Flags().GetString("field")
	if err != nil {
		return false, "", err
	}
	if !raw {
		if cmd.Flags().Changed("field") {
			return false, "", fmt.Errorf("--field requires --raw")
		}
		return false, "", nil
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return false, "", err
	}
	if jsonOutput {
		return false, "", fmt.Errorf("--raw and --json are mutually exclusive")
	}
	if viper.GetBool("debug") {
		return false, "", fmt.Errorf("--debug cannot be used with --raw")
	}
	if strings.TrimSpace(field) == "" {
		return false, "", fmt.Errorf("--field must not be empty")
	}
	return true, field, nil
}
func extractRawField(metadata, secret map[string]any, field string) (string, error) {
	value, err := (credential.ResourceData{Metadata: metadata, Secret: secret}).Field(field)
	if err != nil || value == nil {
		return "", fmt.Errorf("field %q is absent or is not a string", field)
	}
	return *value, nil
}
