package resource

import (
	"context"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
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
		val, _, err := (*program).ContextEval(ctx, map[string]any{
			"Id":             resource.ID,
			"FolderParentID": resource.FolderParentID,
			"Name":           resource.Name,
			"Username":       resource.Username,
			"URI":            resource.URI,
			"Password": func() ref.Val {
				_, _, _, _, pass, _, err := helper.GetResource(ctx, client, resource.ID)
				if err != nil {
					fmt.Printf("Get Resource %v", err)
					return types.String("")
				}
				return types.String(pass)
			},
			"Description": func() ref.Val {
				_, _, _, _, _, descr, err := helper.GetResource(ctx, client, resource.ID)
				if err != nil {
					fmt.Printf("Get Resource %v", err)
					return types.String("")
				}
				return types.String(descr)
			},
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
