package user

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
)

// UserGetCmd Gets a Passbolt User
var UserGetCmd = makeUserGetCmd()

func makeUserGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "user",
		Short: "Gets a Passbolt User",
		Long:  `Gets a Passbolt User`,
		RunE:  UserGet,
	}
}

func configureGet() {
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

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		role, username, firstname, lastname, err := helper.GetUser(
			ctx,
			client,
			id,
		)
		if err != nil {
			return fmt.Errorf("getting User: %w", err)
		}
		if jsonOutput {
			return util.PrintJSON(UserJSONOutput{
				Username:  &username,
				FirstName: &firstname,
				LastName:  &lastname,
				Role:      &role,
			})
		}
		util.Printf("Username: %v\n", util.Sanitize(username))
		util.Printf("FirstName: %v\n", util.Sanitize(firstname))
		util.Printf("LastName: %v\n", util.Sanitize(lastname))
		util.Printf("Role: %v\n", util.Sanitize(role))
		return nil
	})
}

func init() { configureGet() }
