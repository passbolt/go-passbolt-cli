package user

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
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
	UserGetCmd.Flags().Bool("scim-entry", false, "Include SCIM directory linkage (administrator only)")

	UserGetCmd.MarkFlagRequired("id")
}

type userClientRunner func(*cobra.Command, func(context.Context, *api.Client) error) error

func UserGet(cmd *cobra.Command, args []string) error {
	return userGetWithClient(cmd, args, util.WithClient)
}

func userGetWithClient(cmd *cobra.Command, args []string, runClient userClientRunner) error {
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	scimEntry, err := cmd.Flags().GetBool("scim-entry")
	if err != nil {
		return err
	}
	return runClient(cmd, func(ctx context.Context, client *api.Client) error {
		u, err := client.GetUserWithOptions(ctx, id, &api.GetUserOptions{ContainSCIMEntry: scimEntry})
		if err != nil {
			return fmt.Errorf("getting User: %w", err)
		}
		return printUserDetail(*u, jsonOutput)
	})
}

func printUserDetail(u api.User, jsonOutput bool) error {
	out := userJSONOutput(u)
	if jsonOutput {
		return util.PrintJSON(out)
	}
	util.Printf("ID: %v\n", util.Sanitize(u.ID))
	util.Printf("Username: %v\n", util.Sanitize(u.Username))
	if u.Profile != nil {
		util.Printf("FirstName: %v\n", util.Sanitize(u.Profile.FirstName))
		util.Printf("LastName: %v\n", util.Sanitize(u.Profile.LastName))
	}
	if u.Role != nil {
		util.Printf("Role: %v\n", util.Sanitize(u.Role.Name))
	}
	util.Printf("Active: %t\n", u.Active)
	util.Printf("Deleted: %t\n", u.Deleted)
	util.Printf("Disabled: %t\n", u.Disabled != nil)
	if u.IsMFAEnabled != nil {
		util.Printf("MFAEnabled: %t\n", *u.IsMFAEnabled)
	}
	if u.ScimEntry != nil {
		util.Printf("SCIMEntry: %v\n", util.Sanitize(string(u.ScimEntry)))
	}
	return nil
}
