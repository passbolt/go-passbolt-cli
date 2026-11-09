package resource

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

// ResourceDeleteCmd Deletes a Resource
var ResourceDeleteCmd = makeResourceDeleteCmd()

func makeResourceDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resource",
		Short: "Deletes a Passbolt Resource",
		Long:  `Deletes a Passbolt Resource`,
		RunE:  ResourceDelete,
	}
}

func ResourceDelete(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	if id == "" {
		return util.ErrNoID
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		if err := client.DeleteResource(ctx, id); err != nil {
			return fmt.Errorf("deleting Resource %s: %w", id, err)
		}
		return nil
	})
}
