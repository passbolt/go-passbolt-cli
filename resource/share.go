package resource

import (
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// ResourceShareCmd Shares a Passbolt Resource
var ResourceShareCmd = &cobra.Command{
	Use:   "resource",
	Short: "Shares a Passbolt Resource",
	Long:  `Shares a Passbolt Resource`,
	RunE:  ResourceShare,
}

func init() {
	ResourceShareCmd.Flags().String("id", "", "id of Resource to Share")
	ResourceShareCmd.Flags().IntP("type", "t", 1, "Permission Type (1 Read Only, 7 Can Update, 15 Owner)")
	ResourceShareCmd.Flags().StringArrayP("user", "u", []string{}, "User id's to share with")
	ResourceShareCmd.Flags().StringArrayP("group", "g", []string{}, "Group id's to share with")

	ResourceShareCmd.MarkFlagRequired("id")
	ResourceShareCmd.MarkFlagRequired("type")
}

func ResourceShare(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	pType, err := cmd.Flags().GetInt("type")
	if err != nil {
		return err
	}
	users, err := cmd.Flags().GetStringArray("user")
	if err != nil {
		return err
	}
	groups, err := cmd.Flags().GetStringArray("group")
	if err != nil {
		return err
	}

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	err = helper.ShareResourceWithUsersAndGroups(
		ctx,
		client,
		id,
		users,
		groups,
		pType,
	)
	if err != nil {
		return fmt.Errorf("Sharing Resource: %w", err)
	}
	return nil
}
