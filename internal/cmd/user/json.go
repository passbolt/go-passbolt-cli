package user

import (
	"encoding/json"
	"time"

	"github.com/passbolt/go-passbolt/api"
)

type UserJSONOutput struct {
	IsMFAEnabled      *bool           `json:"is_mfa_enabled,omitempty"`
	ScimEntry         json.RawMessage `json:"scim_entry,omitempty"`
	ID                *string         `json:"id,omitempty"`
	Username          *string         `json:"username,omitempty"`
	FirstName         *string         `json:"first_name,omitempty"`
	LastName          *string         `json:"last_name,omitempty"`
	Role              *string         `json:"role,omitempty"`
	CreatedTimestamp  *time.Time      `json:"created_timestamp,omitempty"`
	ModifiedTimestamp *time.Time      `json:"modified_timestamp,omitempty"`
	// Lifecycle flags use non-pointer bool without omitempty so that `false`
	// values reach JSON output and remain distinguishable from "absent" when
	// users do --filter 'active == false'.
	Active   bool `json:"active"`
	Deleted  bool `json:"deleted"`
	Disabled bool `json:"disabled"`
}

// userJSONOutput preserves account state while tolerating omitted optional associations.
func userJSONOutput(u api.User) UserJSONOutput {
	out := UserJSONOutput{ID: &u.ID, Username: &u.Username, Active: u.Active, Deleted: u.Deleted, Disabled: u.Disabled != nil, IsMFAEnabled: u.IsMFAEnabled, ScimEntry: u.ScimEntry}
	if u.Profile != nil {
		out.FirstName, out.LastName = &u.Profile.FirstName, &u.Profile.LastName
	}
	if u.Role != nil {
		out.Role = &u.Role.Name
	}
	if u.Created != nil {
		out.CreatedTimestamp = &u.Created.Time
	}
	if u.Modified != nil {
		out.ModifiedTimestamp = &u.Modified.Time
	}
	return out
}
