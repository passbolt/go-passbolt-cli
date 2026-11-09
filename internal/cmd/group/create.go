package group

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// GroupCreateCmd Creates a Passbolt Group
var GroupCreateCmd = makeGroupCreateCmd()

func makeGroupCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "group",
		Short: "Creates a Passbolt Group",
		Long:  `Creates a Passbolt Group and Returns the Groups ID`,
		RunE:  GroupCreate,
	}
}

func configureCreate() {
	GroupCreateCmd.Flags().StringP("name", "n", "", "Group Name")

	GroupCreateCmd.Flags().StringArrayP("user", "u", []string{}, "Users to Add to Group")
	GroupCreateCmd.Flags().StringArrayP("manager", "m", []string{}, "Managers to Add to Group (atleast 1 is required)")

	GroupCreateCmd.MarkFlagRequired("name")
	GroupCreateCmd.MarkFlagRequired("manager")
}

func GroupCreate(cmd *cobra.Command, args []string) error {
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
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	ops := []helper.GroupMembershipOperation{}
	for _, user := range users {
		ops = append(ops, helper.GroupMembershipOperation{
			UserID:         user,
			IsGroupManager: false,
		})
	}
	for _, manager := range managers {
		ops = append(ops, helper.GroupMembershipOperation{
			UserID:         manager,
			IsGroupManager: true,
		})
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		id, err := helper.CreateGroup(
			ctx,
			client,
			name,
			ops,
		)
		if err != nil {
			return fmt.Errorf("creating Group: %w", err)
		}

		if jsonOutput {
			return util.PrintJSON(map[string]string{"id": id})
		}
		util.Printf("GroupID: %v\n", util.Sanitize(id))
		return nil
	})
}

func init() { configureCreate() }
