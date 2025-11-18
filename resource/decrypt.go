package resource

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
)

// decryptedResource holds the decrypted fields from a resource
type decryptedResource struct {
	name        string
	username    string
	uri         string
	password    string
	description string
}

// decryptResource decrypts a resource's secret and returns the decrypted fields.
// It uses the provided resourceTypeCache to avoid fetching the same ResourceType multiple times.
// If the resource has pre-populated fields (Passbolt v3/v4), it uses those when secrets aren't needed.
func decryptResource(
	ctx context.Context,
	client *api.Client,
	resource api.Resource,
	needsDecryption bool,
	resourceTypeCache map[string]*api.ResourceType,
) (decryptedResource, error) {
	result := decryptedResource{
		name:        resource.Name,
		username:    resource.Username,
		uri:         resource.URI,
		description: resource.Description,
	}

	// If we don't need decryption or no secrets available, return existing fields
	if !needsDecryption || len(resource.Secrets) == 0 {
		return result, nil
	}

	// Check cache first
	rType, exists := resourceTypeCache[resource.ResourceTypeID]
	if !exists {
		var err error
		rType, err = client.GetResourceType(ctx, resource.ResourceTypeID)
		if err != nil {
			return result, fmt.Errorf("Get ResourceType: %w", err)
		}
		resourceTypeCache[resource.ResourceTypeID] = rType
	}

	// Decrypt using the secret
	_, name, username, uri, password, description, err := helper.GetResourceFromData(
		client,
		resource,
		resource.Secrets[0],
		*rType,
	)
	if err != nil {
		return result, fmt.Errorf("Decrypt Resource: %w", err)
	}

	result.name = name
	result.username = username
	result.uri = uri
	result.password = password
	result.description = description

	return result, nil
}
