package user

import (
	"encoding/json"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// UserCreateCmd Creates a Passbolt User
var UserCreateCmd = &cobra.Command{
	Use:   "user",
	Short: "Creates a Passbolt User",
	Long:  `Creates a Passbolt User and Returns the Users ID`,
	RunE:  UserCreate,
}

func init() {
	UserCreateCmd.Flags().StringP("username", "u", "", "Username (needs to be a email address)")
	UserCreateCmd.Flags().StringP("firstname", "f", "", "First Name")
	UserCreateCmd.Flags().StringP("lastname", "l", "", "Last Name")
	UserCreateCmd.Flags().StringP("role", "r", "user", "Role of User.\nPossible: user, admin")

	UserCreateCmd.MarkFlagRequired("username")
	UserCreateCmd.MarkFlagRequired("firstname")
	UserCreateCmd.MarkFlagRequired("lastname")
}

func UserCreate(cmd *cobra.Command, args []string) error {
	username, err := cmd.Flags().GetString("username")
	if err != nil {
		return err
	}
	firstname, err := cmd.Flags().GetString("firstname")
	if err != nil {
		return err
	}
	lastname, err := cmd.Flags().GetString("lastname")
	if err != nil {
		return err
	}
	role, err := cmd.Flags().GetString("role")
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

	id, err := helper.CreateUser(
		ctx,
		client,
		role,
		username,
		firstname,
		lastname,
	)
	if err != nil {
		return fmt.Errorf("Creating User: %w", err)
	}

	if jsonOutput {
		jsonId, err := json.MarshalIndent(
			map[string]string{"id": id},
			"",
			"  ",
		)
		if err != nil {
			return fmt.Errorf("Marshalling Json: %w", err)
		}
		fmt.Println(string(jsonId))
	} else {
		fmt.Printf("UserID: %v\n", id)
	}
	return nil
}
