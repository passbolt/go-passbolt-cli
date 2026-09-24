package user

import (
	"strconv"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
)

var userColumns = util.NewColumnRegistry("users", []util.ColumnSpec[api.User]{
	{
		Name:         "id",
		Aliases:      []string{"ID"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(u api.User) any { return u.ID },
		TableValue:   func(u api.User) string { return u.ID },
	},
	{
		Name:         "username",
		Aliases:      []string{"Username"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(u api.User) any { return u.Username },
		TableValue:   func(u api.User) string { return u.Username },
	},
	{
		Name:         "first_name",
		Aliases:      []string{"FirstName", "firstname"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(u api.User) any { return u.Profile.FirstName },
		TableValue:   func(u api.User) string { return u.Profile.FirstName },
	},
	{
		Name:         "last_name",
		Aliases:      []string{"LastName", "lastname"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(u api.User) any { return u.Profile.LastName },
		TableValue:   func(u api.User) string { return u.Profile.LastName },
	},
	{
		Name:         "role",
		Aliases:      []string{"Role"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(u api.User) any { return u.Role.Name },
		TableValue:   func(u api.User) string { return u.Role.Name },
	},
	{
		Name:         "created_timestamp",
		Aliases:      []string{"CreatedTimestamp", "createdtimestamp"},
		DefaultTable: false,
		CelType:      cel.TimestampType,
		CelValue:     func(u api.User) any { return u.Created.Time },
		TableValue:   func(u api.User) string { return u.Created.Format(time.RFC3339) },
	},
	{
		Name:         "modified_timestamp",
		Aliases:      []string{"ModifiedTimestamp", "modifiedtimestamp"},
		DefaultTable: false,
		CelType:      cel.TimestampType,
		CelValue:     func(u api.User) any { return u.Modified.Time },
		TableValue:   func(u api.User) string { return u.Modified.Format(time.RFC3339) },
	},
	{
		Name:         "active",
		Aliases:      []string{"Active"},
		DefaultTable: false,
		CelType:      cel.BoolType,
		CelValue:     func(u api.User) any { return u.Active },
		TableValue:   func(u api.User) string { return strconv.FormatBool(u.Active) },
	},
	{
		Name:         "deleted",
		Aliases:      []string{"Deleted"},
		DefaultTable: false,
		CelType:      cel.BoolType,
		CelValue:     func(u api.User) any { return u.Deleted },
		TableValue:   func(u api.User) string { return strconv.FormatBool(u.Deleted) },
	},
	{
		// Derived from the nullable *Time field: User.Disabled non-nil ⇒
		// the user is currently disabled. Bool semantics match how users
		// reason about the flag ("is this user disabled?"); an audit-friendly
		// disabled_at timestamp can be added later if needed.
		Name:         "disabled",
		Aliases:      []string{"Disabled"},
		DefaultTable: false,
		CelType:      cel.BoolType,
		CelValue:     func(u api.User) any { return u.Disabled != nil },
		TableValue:   func(u api.User) string { return strconv.FormatBool(u.Disabled != nil) },
	},
})
