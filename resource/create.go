package resource

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
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

	ResourceCreateCmd.MarkFlagRequired("name")
	ResourceCreateCmd.MarkFlagRequired("password")
}

func ResourceCreate(cmd *cobra.Command, args []string) error {
	folderParentID, err := cmd.Flags().GetString("folderParentID")
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

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	id, err := helper.CreateResource(
		ctx,
		client,
		folderParentID,
		name,
		username,
		uri,
		password,
		description,
	)
	if err != nil {
		return fmt.Errorf("Creating Resource: %w", err)
	}

	fmt.Printf("ResourceID: %v\n", id)
	return nil
}
