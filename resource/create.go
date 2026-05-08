package resource

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// ResourceCreateCmd Creates a Passbolt Resource
var ResourceCreateCmd = &cobra.Command{
	Use:   "resource",
	Short: "Creates a Passbolt Resource",
	Long:  `Creates a Passbolt Resource and Returns the Resources ID`,
	RunE:  ResourceCreate,
}

func init() {
	ResourceCreateCmd.Flags().StringP("name", "n", "", "Resource Name")
	ResourceCreateCmd.Flags().StringP("username", "u", "", "Resource Username")
	ResourceCreateCmd.Flags().String("uri", "", "Resource URI")
	ResourceCreateCmd.Flags().StringP("password", "p", "", "Resource Password")
	ResourceCreateCmd.Flags().StringP("description", "d", "", "Resource Description")
	ResourceCreateCmd.Flags().StringP("folderParentID", "f", "", "Folder in which to create the Resource")
	ResourceCreateCmd.Flags().String("expiry", "", "Expiry as RFC3339 (e.g. 2025-12-31T23:59:59Z) or Go duration (e.g. 48h, 30m)")
	ResourceCreateCmd.Flags().String("type", "", "Resource type slug (e.g. v5-default, password-and-description, v5-custom-fields)")
	ResourceCreateCmd.Flags().StringArray("field", []string{}, "Metadata field as key=value (repeatable; JSON values like [\"a\"] are parsed automatically)")
	ResourceCreateCmd.Flags().StringArray("secret-field", []string{}, "Secret field as key=value (repeatable; JSON values are parsed automatically)")
}

func ResourceCreate(cmd *cobra.Command, args []string) error {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	username, err := cmd.Flags().GetString("username")
	if err != nil {
		return err
	}
	uri, err := cmd.Flags().GetString("uri")
	if err != nil {
		return err
	}
	password, err := cmd.Flags().GetString("password")
	if err != nil {
		return err
	}
	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return err
	}
	folderParentID, err := cmd.Flags().GetString("folderParentID")
	if err != nil {
		return err
	}
	expiry, err := cmd.Flags().GetString("expiry")
	if err != nil {
		return err
	}
	resourceType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}
	fields, err := cmd.Flags().GetStringArray("field")
	if err != nil {
		return err
	}
	secretFields, err := cmd.Flags().GetStringArray("secret-field")
	if err != nil {
		return err
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	useGeneric := resourceType != "" || len(fields) > 0 || len(secretFields) > 0

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	var id string

	if useGeneric {
		// Generic path: use CreateResourceGeneric with field maps
		metadataFields := map[string]any{}
		secretFieldsMap := map[string]any{}

		// Map standard flags to field maps
		if name != "" {
			metadataFields["name"] = name
		}
		if username != "" {
			metadataFields["username"] = username
		}
		if uri != "" {
			metadataFields["uri"] = uri
		}
		if description != "" {
			metadataFields["description"] = description
		}
		if password != "" {
			secretFieldsMap["password"] = password
		}

		// Parse --field flags
		for _, f := range fields {
			k, v, err := parseKeyValue(f)
			if err != nil {
				return fmt.Errorf("invalid --field: %w", err)
			}
			metadataFields[k] = v
		}

		// Parse --secret-field flags
		for _, f := range secretFields {
			k, v, err := parseKeyValue(f)
			if err != nil {
				return fmt.Errorf("invalid --secret-field: %w", err)
			}
			secretFieldsMap[k] = v
		}

		if resourceType == "" {
			if client.MetadataTypeSettings().DefaultResourceType == api.PassboltAPIVersionTypeV5 {
				resourceType = "v5-default"
			} else {
				resourceType = "password-and-description"
			}
		}

		id, err = helper.CreateResourceGeneric(ctx, client, resourceType, folderParentID, metadataFields, secretFieldsMap)
	} else {
		// Legacy path: use standard CreateResource
		if name == "" {
			return fmt.Errorf("required flag \"name\" not set")
		}
		if password == "" {
			return fmt.Errorf("required flag \"password\" not set")
		}
		id, err = helper.CreateResource(ctx, client, folderParentID, name, username, uri, password, description)
	}

	if err != nil {
		return fmt.Errorf("creating resource: %w", err)
	}

	if expiry != "" {
		if err := SetResourceExpiry(ctx, client, id, expiry); err != nil {
			return err
		}
	}

	if jsonOutput {
		jsonID, err := json.MarshalIndent(map[string]string{"id": id}, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling json: %w", err)
		}
		fmt.Println(string(jsonID))
	} else {
		fmt.Printf("ResourceID: %v\n", id)
	}
	return nil
}

// parseKeyValue parses a "key=value" string. If the value looks like JSON
// (starts with [ or {), it is decoded into the appropriate Go type so that
// it is serialized correctly when marshaled back to JSON.
func parseKeyValue(s string) (string, any, error) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("expected key=value, got %q", s)
	}
	key := parts[0]
	val := parts[1]

	trimmed := strings.TrimSpace(val)
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		var parsed any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			return key, parsed, nil
		}
	}
	return key, val, nil
}
