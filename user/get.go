package user

import (
	"encoding/json"
	"fmt"

	"al.essio.dev/pkg/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// UserGetCmd Gets a Passbolt User
var UserGetCmd = &cobra.Command{
	Use:   "user",
	Short: "Gets a Passbolt User",
	Long:  `Gets a Passbolt User`,
	RunE:  UserGet,
}

func init() {
	UserGetCmd.Flags().String("id", "", "id of User to Get")

	UserGetCmd.MarkFlagRequired("id")
}

func UserGet(cmd *cobra.Command, args []string) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	role, username, firstname, lastname, err := helper.GetUser(
		ctx,
		client,
		id,
	)
	if err != nil {
		return fmt.Errorf("Getting User: %w", err)
	}
	if jsonOutput {
		jsonUser, err := json.MarshalIndent(UserJsonOutput{
			Username:  &username,
			FirstName: &firstname,
			LastName:  &lastname,
			Role:      &role,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonUser))
	} else {
		fmt.Printf("Username: %v\n", shellescape.StripUnsafe(username))
		fmt.Printf("FirstName: %v\n", shellescape.StripUnsafe(firstname))
		fmt.Printf("LastName: %v\n", shellescape.StripUnsafe(lastname))
		fmt.Printf("Role: %v\n", shellescape.StripUnsafe(role))
	}
	return nil
}
