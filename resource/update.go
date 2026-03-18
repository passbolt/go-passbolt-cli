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
	ResourceUpdateCmd.Flags().StringArray("field", []string{}, "Metadata field as key=value (repeatable; JSON values like [\"a\"] are parsed automatically)")
	ResourceUpdateCmd.Flags().StringArray("secret-field", []string{}, "Secret field as key=value (repeatable; JSON values are parsed automatically)")
	ResourceUpdateCmd.MarkFlagRequired("id")
}

func ResourceUpdate(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
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
	expiry, err := cmd.Flags().GetString("expiry")
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
			metadataUpdates["uri"] = uri
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
