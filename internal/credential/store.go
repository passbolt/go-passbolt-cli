package credential

import (
	"context"
	"strings"

	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
)

type ClientStore struct {
	Client  *api.Client
	Approve func(context.Context, Record) error
}

func (s ClientStore) metadata(ctx context.Context, resource api.Resource) (Record, error) {
	record := Record{ID: strings.ToLower(resource.ID), Name: resource.Name, Username: resource.Username}
	metadata := map[string]any{"uri": resource.URI}
	if resource.Metadata != "" {
		kind := resource.ResourceType
		if kind.Slug == "" {
			loaded, err := s.Client.GetResourceType(ctx, resource.ResourceTypeID)
			if err != nil {
				return Record{}, err
			}
			kind = *loaded
		}
		_, fields, _, err := helper.GetResourceFieldMaps(s.Client, resource, api.Secret{}, kind, false)
		if err != nil {
			return Record{}, err
		}
		metadata = fields
		record.Name = helper.GetStringField(fields, "name")
		record.Username = helper.GetStringField(fields, "username")
	}
	if uri := helper.GetStringField(metadata, "uri"); uri != "" {
		record.URLs = append(record.URLs, uri)
	}
	switch uris := metadata["uris"].(type) {
	case []any:
		for _, value := range uris {
			if uri, ok := value.(string); ok && uri != "" {
				record.URLs = append(record.URLs, uri)
			}
		}
	case []string:
		record.URLs = append(record.URLs, uris...)
	}
	return record, nil
}
func (s ClientStore) Records(ctx context.Context) ([]Record, error) {
	resources, err := s.Client.GetResources(ctx, &api.GetResourcesOptions{ContainResourceType: true})
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(resources))
	for _, resource := range resources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		record, err := s.metadata(ctx, resource)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}
func (s ClientStore) Load(ctx context.Context, id string) (ResourceData, error) {
	resource, err := s.Client.GetResource(ctx, id)
	if err != nil {
		return ResourceData{}, err
	}
	record, err := s.metadata(ctx, *resource)
	if err != nil {
		return ResourceData{}, err
	}
	if !strings.EqualFold(resource.ID, id) {
		return ResourceData{}, ErrLookup
	}
	if s.Approve != nil {
		if err := s.Approve(ctx, record); err != nil {
			return ResourceData{}, err
		}
	}
	kind, err := s.Client.GetResourceType(ctx, resource.ResourceTypeID)
	if err != nil {
		return ResourceData{}, err
	}
	secret, err := s.Client.GetSecret(ctx, id)
	if err != nil {
		return ResourceData{}, err
	}
	_, metadata, fields, err := helper.GetResourceFieldMaps(s.Client, *resource, *secret, *kind, true)
	if err != nil {
		return ResourceData{}, err
	}
	if !strings.EqualFold(resource.ID, id) {
		return ResourceData{}, ErrLookup
	}
	return ResourceData{Record: record, Metadata: metadata, Secret: fields}, nil
}
func (s ClientStore) Read(ctx context.Context, id string) (Record, string, error) {
	data, err := s.Load(ctx, id)
	if err != nil {
		return Record{}, "", err
	}
	password, err := data.Password()
	if err != nil {
		return Record{}, "", err
	}
	return data.Record, password, nil
}
