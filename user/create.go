package user

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
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

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	user, err := client.CreateUser(
		ctx,
		api.User{
			Username: username,
			Profile: &api.Profile{
				FirstName: firstname,
				LastName:  lastname,
			},
			Role: &api.Role{
				Name: role,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("Creating User: %w", err)
	}

	if jsonOutput {
		jsonUser, err := json.MarshalIndent(user, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonUser))
	} else {
		fmt.Printf("UserID: %v\n", user.ID)
	}
	return nil
}
