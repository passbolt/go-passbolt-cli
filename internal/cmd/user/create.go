package user

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// UserCreateCmd Creates a Passbolt User
var UserCreateCmd = makeUserCreateCmd()

func makeUserCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "user",
		Short: "Creates a Passbolt User",
		Long:  `Creates a Passbolt User and Returns the Users ID`,
		RunE:  UserCreate,
	}
}

func configureCreate() {
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

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		id, err := helper.CreateUser(
			ctx,
			client,
			role,
			username,
			firstname,
			lastname,
		)
		if err != nil {
			return fmt.Errorf("creating User: %w", err)
		}

		if jsonOutput {
			return util.PrintJSON(map[string]string{"id": id})
		}
		util.Printf("UserID: %v\n", util.Sanitize(id))
		return nil
	})
}

func init() { configureCreate() }
