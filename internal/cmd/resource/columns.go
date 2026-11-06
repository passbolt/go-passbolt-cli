package resource

import (
	"encoding/json"
	"strconv"
	"time"

	"cel.dev/cel-go/cel"
	"github.com/passbolt/go-passbolt-cli/internal/util"
)

func marshalMapForTable(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

// resourceColumns is the registry of --column / --filter attributes on
// Resource. requiresSecrets marks columns whose value depends on the
// (expensive) secret decryption path — used to decide whether to fetch secrets
// up-front.
var resourceColumns = util.NewColumnRegistry("resources", []util.ColumnSpec[decryptedResource]{
	{
		Name:         "id",
		Aliases:      []string{"ID"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(d decryptedResource) any { return d.resource.ID },
		TableValue:   func(d decryptedResource) string { return d.resource.ID },
	},
	{
		Name:         "folder_parent_id",
		Aliases:      []string{"FolderParentID", "folderparentid"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(d decryptedResource) any { return d.resource.FolderParentID },
		TableValue:   func(d decryptedResource) string { return d.resource.FolderParentID },
	},
	{
		Name:         "name",
		Aliases:      []string{"Name"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(d decryptedResource) any { return d.name },
		TableValue:   func(d decryptedResource) string { return d.name },
	},
	{
		Name:         "username",
		Aliases:      []string{"Username"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(d decryptedResource) any { return d.username },
		TableValue:   func(d decryptedResource) string { return d.username },
	},
	{
		Name:         "uri",
		Aliases:      []string{"URI"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(d decryptedResource) any { return d.uri },
		TableValue:   func(d decryptedResource) string { return d.uri },
	},
	{
		Name:            "password",
		Aliases:         []string{"Password"},
		DefaultTable:    false,
		RequiresSecrets: true,
		CelType:         cel.StringType,
		CelValue:        func(d decryptedResource) any { return d.password },
		TableValue:      func(d decryptedResource) string { return d.password },
	},
	{
		Name:            "description",
		Aliases:         []string{"Description"},
		DefaultTable:    false,
		RequiresSecrets: true,
		CelType:         cel.StringType,
		CelValue:        func(d decryptedResource) any { return d.description },
		TableValue:      func(d decryptedResource) string { return d.description },
	},
	{
		Name:         "created_timestamp",
		Aliases:      []string{"CreatedTimestamp", "createdtimestamp"},
		DefaultTable: false,
		CelType:      cel.TimestampType,
		CelValue:     func(d decryptedResource) any { return d.resource.Created.Time },
		TableValue:   func(d decryptedResource) string { return d.resource.Created.Format(time.RFC3339) },
	},
	{
		Name:         "modified_timestamp",
		Aliases:      []string{"ModifiedTimestamp", "modifiedtimestamp"},
		DefaultTable: false,
		CelType:      cel.TimestampType,
		CelValue:     func(d decryptedResource) any { return d.resource.Modified.Time },
		TableValue:   func(d decryptedResource) string { return d.resource.Modified.Format(time.RFC3339) },
	},
	{
		Name:         "metadata",
		Aliases:      []string{"Metadata"},
		DefaultTable: false,
		CelType:      cel.MapType(cel.StringType, cel.DynType),
		CelValue: func(d decryptedResource) any {
			if d.metadataFields == nil {
				return map[string]any{}
			}
			return d.metadataFields
		},
		TableValue: func(d decryptedResource) string { return marshalMapForTable(d.metadataFields) },
	},
	{
		Name:            "secret",
		Aliases:         []string{"Secret"},
		DefaultTable:    false,
		RequiresSecrets: true,
		CelType:         cel.MapType(cel.StringType, cel.DynType),
		CelValue: func(d decryptedResource) any {
			if d.secretFields == nil {
				return map[string]any{}
			}
			return d.secretFields
		},
		TableValue: func(d decryptedResource) string { return marshalMapForTable(d.secretFields) },
	},
	{
		Name:         "deleted",
		Aliases:      []string{"Deleted"},
		DefaultTable: false,
		CelType:      cel.BoolType,
		CelValue:     func(d decryptedResource) any { return d.resource.Deleted },
		TableValue:   func(d decryptedResource) string { return strconv.FormatBool(d.resource.Deleted) },
	},
	{
		// Derived bool: true iff Resource.Expired is set. Pairs with the
		// expired_at column below which exposes the actual timestamp.
		Name:         "expired",
		Aliases:      []string{"Expired"},
		DefaultTable: false,
		CelType:      cel.BoolType,
		CelValue:     func(d decryptedResource) any { return d.resource.Expired != nil },
		TableValue:   func(d decryptedResource) string { return strconv.FormatBool(d.resource.Expired != nil) },
	},
	{
		// Nullable timestamp: zero time when not expired. CEL users compare
		// against timestamp() literals; table renders RFC3339 or empty.
		Name:         "expired_at",
		Aliases:      []string{"ExpiredAt", "expiredat"},
		DefaultTable: false,
		CelType:      cel.TimestampType,
		CelValue: func(d decryptedResource) any {
			if d.resource.Expired == nil {
				return time.Time{}
			}
			return d.resource.Expired.Time
		},
		TableValue: func(d decryptedResource) string {
			if d.resource.Expired == nil {
				return ""
			}
			return d.resource.Expired.Format(time.RFC3339)
		},
	},
	{
		Name:         "resource_type_id",
		Aliases:      []string{"ResourceTypeID", "resourcetypeid"},
		DefaultTable: false,
		CelType:      cel.StringType,
		CelValue:     func(d decryptedResource) any { return d.resource.ResourceTypeID },
		TableValue:   func(d decryptedResource) string { return d.resource.ResourceTypeID },
	},
})
