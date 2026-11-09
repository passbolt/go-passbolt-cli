package util

import (
	"context"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
)

// ResourceApproval is scoped to a serialized agent invocation and called before
// requesting a resource secret. Names are metadata, not decrypted passwords.
var ResourceApproval func(context.Context, string, string) error

func ApproveResource(ctx context.Context, id, name string) error {
	if ResourceApproval != nil {
		return ResourceApproval(ctx, id, name)
	}
	return nil
}

func ApproveResourceRead(ctx context.Context, client *api.Client, id string) error {
	if ResourceApproval == nil {
		return nil
	}
	resource, err := client.GetResource(ctx, id)
	if err != nil {
		return err
	}
	kind, err := client.GetResourceType(ctx, resource.ResourceTypeID)
	if err != nil {
		return err
	}
	_, metadata, _, err := helper.GetResourceFieldMaps(client, *resource, api.Secret{}, *kind, false)
	if err != nil {
		return err
	}
	return ApproveResource(ctx, resource.ID, helper.GetStringField(metadata, "name"))
}
