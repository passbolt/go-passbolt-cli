package resource

import (
	"context"
	"encoding/json"
	"fmt"

	"al.essio.dev/pkg/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
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
	jsonOutput, err := cmd.Flags().GetBool("json")
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

	if jsonOutput {
		jsonResource, err := json.MarshalIndent(ResourceJsonOutput{
			FolderParentID: &folderParentID,
			Name:           &name,
			Username:       &username,
			URI:            &uri,
			Password:       &password,
			Description:    &description,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonResource))
	} else {
		fmt.Printf("FolderParentID: %v\n", folderParentID)
		fmt.Printf("Name: %v\n", shellescape.StripUnsafe(name))
		fmt.Printf("Username: %v\n", shellescape.StripUnsafe(username))
		fmt.Printf("URI: %v\n", shellescape.StripUnsafe(uri))
		fmt.Printf("Password: %v\n", shellescape.StripUnsafe(password))
		fmt.Printf("Description: %v\n", shellescape.StripUnsafe(description))
	}
	return nil
}
