package user

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alessio/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
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

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	user, err := client.GetUser(ctx, id)
	if err != nil {
		return fmt.Errorf("Getting User: %w", err)
	}
	if jsonOutput {
		jsonUser, err := json.MarshalIndent(*user, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonUser))
	} else {
		fmt.Printf("Username: %v\n", shellescape.StripUnsafe(user.Username))
		fmt.Printf("FirstName: %v\n", shellescape.StripUnsafe(user.Profile.FirstName))
		fmt.Printf("LastName: %v\n", shellescape.StripUnsafe(user.Profile.LastName))
		fmt.Printf("Role: %v\n", shellescape.StripUnsafe(user.Role.Name))
	}

	return nil
}
