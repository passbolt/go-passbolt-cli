package folder

import "time"

type FolderJsonOutput struct {
	ID                *string    `json:"id,omitempty"`
	FolderParentID    *string    `json:"folder_parent_id,omitempty"`
	Name              *string    `json:"name,omitempty"`
	CreatedTimestamp  *time.Time `json:"created_timestamp,omitempty"`
	ModifiedTimestamp *time.Time `json:"modified_timestamp,omitempty"`
}
