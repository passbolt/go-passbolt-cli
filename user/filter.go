package user

import (
	"context"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
)

// Environments for CEl
var celEnvOptions = []cel.EnvOption{
	cel.Variable("ID", cel.StringType),
	cel.Variable("Username", cel.StringType),
	cel.Variable("FirstName", cel.StringType),
	cel.Variable("LastName", cel.StringType),
	cel.Variable("Role", cel.StringType),
	cel.Variable("CreatedTimestamp", cel.TimestampType),
	cel.Variable("ModifiedTimestamp", cel.TimestampType),
}

// Filters the slice users by invoke CEL program for each user
func filterUsers(users *[]api.User, celCmd string, ctx context.Context) ([]api.User, error) {
	if celCmd == "" {
		return *users, nil
	}

	program, err := util.InitCELProgram(celCmd, celEnvOptions...)
	if err != nil {
		return nil, err
	}

	filteredUsers := []api.User{}
	for _, user := range *users {
		val, _, err := (*program).ContextEval(ctx, map[string]any{
			"ID":                user.ID,
			"Username":          user.Username,
			"FirstName":         user.Profile.FirstName,
			"LastName":          user.Profile.LastName,
			"Role":              user.Role.Name,
			"CreatedTimestamp":  user.Created.Time,
			"ModifiedTimestamp": user.Modified.Time,
		})

		if err != nil {
			return nil, err
		}

		if val.Value() == true {
			filteredUsers = append(filteredUsers, user)
		}
	}

	if len(filteredUsers) == 0 {
		return nil, fmt.Errorf("No such users found with filter %v!", celCmd)
	}

	return filteredUsers, nil
}
