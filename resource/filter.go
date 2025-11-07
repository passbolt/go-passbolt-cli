package resource

import (
	"context"
	"fmt"
	"strings"

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

	// Check if filter uses encrypted fields (Name, Username, URI, Password, Description)
	// We do a simple string check - if any of these fields appear in the filter, we need to decrypt
	needsDecryption := false
	encryptedFields := []string{"Name", "Username", "URI", "Password", "Description"}
	for _, field := range encryptedFields {
		if strings.Contains(celCmd, field) {
			needsDecryption = true
			break
		}
	}

	// Cache resource types to avoid fetching the same type repeatedly
	resourceTypeCache := make(map[string]*api.ResourceType)

	filteredResources := []api.Resource{}
	for _, resource := range *resources {
		// Decrypt resource if filter needs encrypted fields
		decrypted, err := decryptResource(ctx, client, resource, needsDecryption, resourceTypeCache)
		if err != nil {
			return nil, err
		}

		// Fallback: fetch individually if fields are empty and no secrets included
		if needsDecryption && len(resource.Secrets) == 0 {
			if decrypted.name == "" || decrypted.username == "" || decrypted.uri == "" || decrypted.description == "" {
				_, decrypted.name, decrypted.username, decrypted.uri, decrypted.password, decrypted.description, err = helper.GetResource(ctx, client, resource.ID)
				if err != nil {
					return nil, fmt.Errorf("Get Resource: %w", err)
				}
			}
		}

		val, _, err := (*program).ContextEval(ctx, map[string]any{
			"ID":                resource.ID,
			"FolderParentID":    resource.FolderParentID,
			"Name":              decrypted.name,
			"Username":          decrypted.username,
			"URI":               decrypted.uri,
			"Password":          decrypted.password,
			"Description":       decrypted.description,
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
