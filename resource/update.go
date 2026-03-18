package resource

import (
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// ResourceUpdateCmd Updates a Passbolt Resource
var ResourceUpdateCmd = &cobra.Command{
	Use:   "resource",
	Short: "Updates a Passbolt Resource",
	Long:  `Updates a Passbolt Resource`,
	RunE:  ResourceUpdate,
}

func init() {
	ResourceUpdateCmd.Flags().String("id", "", "id of Resource to Update")
	ResourceUpdateCmd.Flags().StringP("name", "n", "", "Resource Name")
	ResourceUpdateCmd.Flags().StringP("username", "u", "", "Resource Username")
	ResourceUpdateCmd.Flags().String("uri", "", "Resource URI")
	ResourceUpdateCmd.Flags().StringP("password", "p", "", "Resource Password")
	ResourceUpdateCmd.Flags().StringP("description", "d", "", "Resource Description")
	ResourceUpdateCmd.Flags().String("expiry", "", "Expiry as RFC3339 (e.g. 2025-12-31T23:59:59Z), duration (e.g. 7d, 12h), or 'none' to clear")
	ResourceUpdateCmd.Flags().StringArray("field", []string{}, "Metadata field as key=value (repeatable)")
	ResourceUpdateCmd.Flags().StringArray("secret-field", []string{}, "Secret field as key=value (repeatable)")
	ResourceUpdateCmd.MarkFlagRequired("id")
}

func ResourceUpdate(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	name, _ := cmd.Flags().GetString("name")
	username, _ := cmd.Flags().GetString("username")
	uri, _ := cmd.Flags().GetString("uri")
	password, _ := cmd.Flags().GetString("password")
	description, _ := cmd.Flags().GetString("description")
	expiry, _ := cmd.Flags().GetString("expiry")
	fields, _ := cmd.Flags().GetStringArray("field")
	secretFields, _ := cmd.Flags().GetStringArray("secret-field")

	useGeneric := len(fields) > 0 || len(secretFields) > 0

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	if useGeneric {
		// Generic path: use UpdateResourceGeneric with field maps
		metadataUpdates := map[string]any{}
		secretUpdates := map[string]any{}

		if name != "" {
			metadataUpdates["name"] = name
		}
		if username != "" {
			metadataUpdates["username"] = username
		}
		if uri != "" {
			// Fetch the resource to determine if v5 (needs "uris") or v4 (needs "uri")
			resource, fetchErr := client.GetResource(ctx, id)
			if fetchErr != nil {
				return fmt.Errorf("getting resource: %w", fetchErr)
			}
			if resource.Metadata != "" {
				metadataUpdates["uris"] = []string{uri}
			} else {
				metadataUpdates["uri"] = uri
			}
		}
		if description != "" {
			metadataUpdates["description"] = description
		}
		if password != "" {
			secretUpdates["password"] = password
		}

		for _, f := range fields {
			k, v, parseErr := parseKeyValue(f)
			if parseErr != nil {
				return fmt.Errorf("invalid --field: %w", parseErr)
			}
			metadataUpdates[k] = v
		}
		for _, f := range secretFields {
			k, v, parseErr := parseKeyValue(f)
			if parseErr != nil {
				return fmt.Errorf("invalid --secret-field: %w", parseErr)
			}
			secretUpdates[k] = v
		}

		err = helper.UpdateResourceGeneric(ctx, client, id, metadataUpdates, secretUpdates)
	} else {
		err = helper.UpdateResource(ctx, client, id, name, username, uri, password, description)
	}

	if err != nil {
		return fmt.Errorf("updating resource: %w", err)
	}

	if expiry != "" {
		if err := SetResourceExpiry(ctx, client, id, expiry); err != nil {
			return err
		}
	}
	return nil
}
