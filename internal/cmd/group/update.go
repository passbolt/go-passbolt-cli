package group

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// GroupUpdateCmd Updates a Passbolt Group
var GroupUpdateCmd = makeGroupUpdateCmd()

func makeGroupUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "group",
		Short: "Updates a Passbolt Group",
		Long:  `Updates a Passbolt Group`,
		RunE:  GroupUpdate,
	}
}

func configureUpdate() {
	GroupUpdateCmd.Flags().String("id", "", "id of Group to Update")
	GroupUpdateCmd.Flags().StringP("name", "n", "", "Group Name")

	GroupUpdateCmd.Flags().BoolP("delete", "d", false, "Remove Users/Managers from Group (default is Adding Users/Managers)")

	GroupUpdateCmd.Flags().StringArrayP("user", "u", []string{}, "Users to Add/Remove to/from Group(Including Group Managers)")
	GroupUpdateCmd.Flags().StringArrayP("manager", "m", []string{}, "Managers to Add/Remove to/from Group")

	GroupUpdateCmd.MarkFlagRequired("id")
}

func GroupUpdate(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	delete, err := cmd.Flags().GetBool("delete")
	if err != nil {
		return err
	}
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	users, err := cmd.Flags().GetStringArray("user")
	if err != nil {
		return err
	}
	managers, err := cmd.Flags().GetStringArray("manager")
	if err != nil {
		return err
	}

	ops := []helper.GroupMembershipOperation{}
	for _, user := range users {
		ops = append(ops, helper.GroupMembershipOperation{
			UserID:         user,
			IsGroupManager: false,
			Delete:         delete,
		})
	}
	for _, manager := range managers {
		ops = append(ops, helper.GroupMembershipOperation{
			UserID:         manager,
			IsGroupManager: true,
			Delete:         delete,
		})
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := helper.UpdateGroup(
			ctx,
			client,
			id,
			name,
			ops,
		); err != nil {
			return fmt.Errorf("updating Group: %w", err)
		}
		return nil
	})
}

func init() { configureUpdate() }
