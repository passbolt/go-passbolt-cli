package resource

import "time"

type ResourceJsonOutput struct {
	ID                *string    `json:"id,omitempty"`
	FolderParentID    *string    `json:"folder_parent_id,omitempty"`
	Name              *string    `json:"name,omitempty"`
	Username          *string    `json:"username,omitempty"`
	URI               *string    `json:"uri,omitempty"`
	Password          *string    `json:"password,omitempty"`
	Description       *string    `json:"description,omitempty"`
	CreatedTimestamp  *time.Time `json:"created_timestamp,omitempty"`
	ModifiedTimestamp *time.Time `json:"modified_timestamp,omitempty"`
}

type PermissionJsonOutput struct {
	ID                *string    `json:"id,omitempty"`
	Aco               *string    `json:"aco,omitempty"`
	AcoForeignKey     *string    `json:"aco_foreign_key,omitempty"`
	Aro               *string    `json:"aro,omitempty"`
	AroForeignKey     *string    `json:"aro_foreign_key,omitempty"`
	Type              *int       `json:"type,omitempty"`
	CreatedTimestamp  *time.Time `json:"created_timestamp,omitempty"`
	ModifiedTimestamp *time.Time `json:"modified_timestamp,omitempty"`
}
