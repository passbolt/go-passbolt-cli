package resource

import (
	"context"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
)

// Environments for CEl
var celEnvOptions = []cel.EnvOption{
	cel.Variable("ID", cel.StringType),
	cel.Variable("FolderParentID", cel.StringType),
	cel.Variable("Name", cel.StringType),
	cel.Variable("Username", cel.StringType),
	cel.Variable("URI", cel.StringType),
	cel.Variable("Password", cel.StringType),
	cel.Variable("Description", cel.StringType),
	cel.Variable("CreatedTimestamp", cel.TimestampType),
	cel.Variable("ModifiedTimestamp", cel.TimestampType),
}

// Filters the slice resources by invoke CEL program for each resource
func filterResources(resources *[]api.Resource, celCmd string, ctx context.Context, client *api.Client) ([]api.Resource, error) {
	if celCmd == "" {
		return *resources, nil
	}

	program, err := util.InitCELProgram(celCmd, celEnvOptions...)
	if err != nil {
		return nil, err
	}

	filteredResources := []api.Resource{}
	for _, resource := range *resources {
		// TODO We should decrypt the secret only when required for performance reasonse
		_, name, username, uri, pass, desc, err := helper.GetResource(ctx, client, resource.ID)
		if err != nil {
			return nil, fmt.Errorf("Get Resource %w", err)
		}

		val, _, err := (*program).ContextEval(ctx, map[string]any{
			"Id":                resource.ID,
			"FolderParentID":    resource.FolderParentID,
			"Name":              name,
			"Username":          username,
			"URI":               uri,
			"Password":          pass,
			"Description":       desc,
			"CreatedTimestamp":  resource.Created.Time,
			"ModifiedTimestamp": resource.Modified.Time,
		})

		if err != nil {
			return nil, err
		}

		if val.Value() == true {
			filteredResources = append(filteredResources, resource)
		}
	}

	if len(filteredResources) == 0 {
		return nil, fmt.Errorf("No such Resources found with filter %v!", celCmd)
	}
	return filteredResources, nil
}
