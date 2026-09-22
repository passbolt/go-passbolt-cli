package group

import (
	"strconv"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
)

var groupColumns = util.NewColumnRegistry("groups", []util.ColumnSpec[api.Group]{
	{
		Name:         "id",
		Aliases:      []string{"ID"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(g api.Group) any { return g.ID },
		TableValue:   func(g api.Group) string { return g.ID },
	},
	{
		Name:         "name",
		Aliases:      []string{"Name"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(g api.Group) any { return g.Name },
		TableValue:   func(g api.Group) string { return g.Name },
	},
	{
		Name:         "created_timestamp",
		Aliases:      []string{"CreatedTimestamp", "createdtimestamp"},
		DefaultTable: false,
		CelType:      cel.TimestampType,
		CelValue:     func(g api.Group) any { return g.Created.Time },
		TableValue:   func(g api.Group) string { return g.Created.Format(time.RFC3339) },
	},
	{
		Name:         "modified_timestamp",
		Aliases:      []string{"ModifiedTimestamp", "modifiedtimestamp"},
		DefaultTable: false,
		CelType:      cel.TimestampType,
		CelValue:     func(g api.Group) any { return g.Modified.Time },
		TableValue:   func(g api.Group) string { return g.Modified.Format(time.RFC3339) },
	},
	{
		Name:         "deleted",
		Aliases:      []string{"Deleted"},
		DefaultTable: false,
		CelType:      cel.BoolType,
		CelValue:     func(g api.Group) any { return g.Deleted },
		TableValue:   func(g api.Group) string { return strconv.FormatBool(g.Deleted) },
	},
	{
		Name:         "user_count",
		Aliases:      []string{"UserCount", "usercount"},
		DefaultTable: false,
		CelType:      cel.IntType,
		CelValue:     func(g api.Group) any { return int64(g.UserCount) },
		TableValue:   func(g api.Group) string { return strconv.Itoa(g.UserCount) },
	},
})
