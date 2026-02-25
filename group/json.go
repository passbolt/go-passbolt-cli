package group

import "time"

type GroupJSONOutput struct {
	ID                *string                         `json:"id,omitempty"`
	Name              *string                         `json:"name,omitempty"`
	Users             []GroupUserMembershipJSONOutput `json:"users,omitempty"`
	CreatedTimestamp  *time.Time                      `json:"created_timestamp,omitempty"`
	ModifiedTimestamp *time.Time                      `json:"modified_timestamp,omitempty"`
}

type GroupUserMembershipJSONOutput struct {
	ID             *string `json:"id,omitempty"`
	Username       *string `json:"username,omitempty"`
	FirstName      *string `json:"first_name,omitempty"`
	LastName       *string `json:"last_name,omitempty"`
	IsGroupManager *bool   `json:"is_group_manager,omitempty"`
}
