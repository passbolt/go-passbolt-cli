package folder

import (
	"strconv"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
)

var folderColumns = util.NewColumnRegistry("folders", []util.ColumnSpec[api.Folder]{
	{
		Name:         "id",
		Aliases:      []string{"ID"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(f api.Folder) any { return f.ID },
		TableValue:   func(f api.Folder) string { return f.ID },
	},
	{
		Name:         "folder_parent_id",
		Aliases:      []string{"FolderParentID", "folderparentid"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(f api.Folder) any { return f.FolderParentID },
		TableValue:   func(f api.Folder) string { return f.FolderParentID },
	},
	{
		Name:         "name",
		Aliases:      []string{"Name"},
		DefaultTable: true,
		CelType:      cel.StringType,
		CelValue:     func(f api.Folder) any { return f.Name },
		TableValue:   func(f api.Folder) string { return f.Name },
	},
	{
		Name:         "created_timestamp",
		Aliases:      []string{"CreatedTimestamp", "createdtimestamp"},
		DefaultTable: false,
		CelType:      cel.TimestampType,
		CelValue:     func(f api.Folder) any { return f.Created.Time },
		TableValue:   func(f api.Folder) string { return f.Created.Format(time.RFC3339) },
	},
	{
		Name:         "modified_timestamp",
		Aliases:      []string{"ModifiedTimestamp", "modifiedtimestamp"},
		DefaultTable: false,
		CelType:      cel.TimestampType,
		CelValue:     func(f api.Folder) any { return f.Modified.Time },
		TableValue:   func(f api.Folder) string { return f.Modified.Format(time.RFC3339) },
	},
	{
		Name:         "personal",
		Aliases:      []string{"Personal"},
		DefaultTable: false,
		CelType:      cel.BoolType,
		CelValue:     func(f api.Folder) any { return f.Personal },
		TableValue:   func(f api.Folder) string { return strconv.FormatBool(f.Personal) },
	},
})
