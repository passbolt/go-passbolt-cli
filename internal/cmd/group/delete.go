package group

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

// GroupDeleteCmd Deletes a Group
var GroupDeleteCmd = makeGroupDeleteCmd()

func makeGroupDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "group",
		Short: "Deletes a Passbolt Group",
		Long:  `Deletes a Passbolt Group`,
		RunE:  GroupDelete,
	}
}

func GroupDelete(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	if id == "" {
		return util.ErrNoID
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := client.DeleteGroup(ctx, id); err != nil {
			return fmt.Errorf("deleting Group %s: %w", id, err)
		}
		return nil
	})
}
