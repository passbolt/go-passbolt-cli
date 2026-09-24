package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/passbolt/go-passbolt/api"
	"github.com/pterm/pterm"
)

// PermissionsToJSONOutput maps API permissions to the wire output struct shared by
// the resource and folder permission commands.
func PermissionsToJSONOutput(permissions []api.Permission) []PermissionJSONOutput {
	out := make([]PermissionJSONOutput, 0, len(permissions))
	for i := range permissions {
		out = append(out, PermissionJSONOutput{
			ID:                &permissions[i].ID,
			Aco:               &permissions[i].ACO,
			AcoForeignKey:     &permissions[i].ACOForeignKey,
			Aro:               &permissions[i].ARO,
			AroForeignKey:     &permissions[i].AROForeignKey,
			Type:              &permissions[i].Type,
			CreatedTimestamp:  &permissions[i].Created.Time,
			ModifiedTimestamp: &permissions[i].Modified.Time,
		})
	}
	return out
}

// PermissionColumns lists the column names accepted by the permission table
// commands (resource/folder permission). Used to build "valid columns" guidance.
var PermissionColumns = []string{
	"ID", "Aco", "AcoForeignKey", "Aro", "AroForeignKey", "Type", "CreatedTimestamp", "ModifiedTimestamp",
}

// Passbolt permission type codes. These are the bitmask-derived values the API
// returns for a permission's Type field.
const (
	permissionRead   = 1
	permissionUpdate = 7
	permissionOwner  = 15
)

// permissionTypeName maps a Passbolt permission type code to its human-readable
// name for table output. Unknown codes fall back to their numeric form so the
// table never hides an unexpected value.
func permissionTypeName(t int) string {
	switch t {
	case permissionRead:
		return "read"
	case permissionUpdate:
		return "update"
	case permissionOwner:
		return "owner"
	default:
		return strconv.Itoa(t)
	}
}

// PrintPermissionTable renders the permission table using the legacy
// lowercased-column switch shared by the resource and folder permission commands.
// An unknown column yields an "unknown Column: %v" error (capitalized, preserving
// the historical message).
func PrintPermissionTable(columns []string, permissions []api.Permission) error {
	data := pterm.TableData{columns}
	for _, p := range permissions {
		entry := make([]string, len(columns))
		for i := range columns {
			v, ok := permissionCell(p, columns[i])
			if !ok {
				return fmt.Errorf("unknown Column: %v", columns[i])
			}
			entry[i] = v
		}
		data = append(data, entry)
	}
	RenderTable(data)
	return nil
}

// permissionCell renders a single permission cell for the given column name. Column
// matching is case-insensitive. The bool reports whether the column is known.
func permissionCell(p api.Permission, column string) (string, bool) {
	switch strings.ToLower(column) {
	case "id":
		return p.ID, true
	case "aco":
		return p.ACO, true
	case "acoforeignkey":
		return p.ACOForeignKey, true
	case "aro":
		return p.ARO, true
	case "aroforeignkey":
		return p.AROForeignKey, true
	case "type":
		return permissionTypeName(p.Type), true
	case "createdtimestamp":
		return p.Created.Format(time.RFC3339), true
	case "modifiedtimestamp":
		return p.Modified.Format(time.RFC3339), true
	default:
		return "", false
	}
}
