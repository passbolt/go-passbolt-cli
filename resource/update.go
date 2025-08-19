package resource

import (
	"context"
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
	ResourceUpdateCmd.Flags().String("expired", "", "Expiry date/time (ISO8601), e.g. 2025-12-31T23:59:59Z; use empty to clear with --clear-expired")
	ResourceUpdateCmd.Flags().Bool("clear-expired", false, "Clear expiry (sets expired to null)")

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
	expired, err := cmd.Flags().GetString("expired")
	if err != nil {
		return err
	}
	clearExpired, err := cmd.Flags().GetBool("clear-expired")
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

	err = helper.UpdateResource(
		ctx,
		client,
		id,
		name,
		username,
		uri,
		password,
		description,
	)
	if err != nil {
		return fmt.Errorf("Updating Resource: %w", err)
	}

	if clearExpired {
		// explicit clear to null
		_, _, err := client.DoCustomRequestAndReturnRawResponse(
			ctx,
			"PUT",
			fmt.Sprintf("resources/%s.json", id),
			"v2",
			map[string]*string{"expired": nil},
			nil,
		)
		if err != nil {
			return fmt.Errorf("Clearing expiry: %w", err)
		}
	} else if expired != "" {
		if err := SetResourceExpiry(ctx, client, id, expired); err != nil {
			return err
		}
	}
	return nil
}
