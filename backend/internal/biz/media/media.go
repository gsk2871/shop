// Package media defines media identities and references shared by all content modules.
package media

import "time"

type Media struct {
	ID             int64      `json:"id,string" gorm:"column:id"`
	Name           string     `json:"name" gorm:"column:name"`
	URL            string     `json:"url" gorm:"column:url"`
	MIME           string     `json:"mime" gorm:"column:mime"`
	MediaType      string     `json:"type" gorm:"column:media_type"`
	Size           int64      `json:"size" gorm:"column:size"`
	Width          *int       `json:"width" gorm:"column:width"`
	Height         *int       `json:"height" gorm:"column:height"`
	Available      bool       `json:"available" gorm:"column:available"`
	StorageRootID  string     `json:"-" gorm:"column:storage_root_id"`
	StorageKey     string     `json:"-" gorm:"column:storage_key"`
	UploadedBy     *int64     `json:"uploadedBy,string" gorm:"column:uploaded_by"`
	CleanupPending bool       `json:"-" gorm:"column:cleanup_pending"`
	CreatedAt      time.Time  `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt      time.Time  `json:"updatedAt" gorm:"column:updated_at"`
	DeletedAt      *time.Time `json:"-" gorm:"column:deleted_at"`
}
type Reference struct {
	ID        int64     `json:"id,string"`
	MediaID   int64     `json:"mediaId,string"`
	OwnerType string    `json:"ownerType"`
	OwnerID   string    `json:"ownerId"`
	Field     string    `json:"field"`
	CreatedAt time.Time `json:"createdAt"`
}
