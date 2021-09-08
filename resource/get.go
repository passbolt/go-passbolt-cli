package resource

import (
	"context"
	"fmt"

	"github.com/speatzle/go-passbolt-cli/util"
	"github.com/speatzle/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// ResourceGetCmd Gets a Passbolt Resource
var ResourceGetCmd = &cobra.Command{
	Use:   "resource",
	Short: "Gets a Passbolt Resource",
	Long:  `Gets a Passbolt Resource`,
	RunE:  ResourceGet,
}

func init() {
	ResourceGetCmd.Flags().String("id", "", "id of Resource to Get")

	ResourceGetCmd.MarkFlagRequired("id")
}

func ResourceGet(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
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

	folderParentID, name, username, uri, password, description, err := helper.GetResource(
		ctx,
		client,
		id,
	)
	if err != nil {
		return fmt.Errorf("Getting Resource: %w", err)
	}
	fmt.Printf("FolderParentID: %v\n", folderParentID)
	fmt.Printf("Name: %v\n", name)
	fmt.Printf("Username: %v\n", username)
	fmt.Printf("URI: %v\n", uri)
	fmt.Printf("Password: %v\n", password)
	fmt.Printf("Description: %v\n", description)
	return nil
}
