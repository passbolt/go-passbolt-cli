package user

import "time"

type UserJsonOutput struct {
	ID                *string    `json:"id,omitempty"`
	Username          *string    `json:"username,omitempty"`
	FirstName         *string    `json:"first_name,omitempty"`
	LastName          *string    `json:"last_name,omitempty"`
	Role              *string    `json:"role,omitempty"`
	CreatedTimestamp  *time.Time `json:"created_timestamp,omitempty"`
	ModifiedTimestamp *time.Time `json:"modified_timestamp,omitempty"`
}
