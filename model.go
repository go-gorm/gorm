package gorm

import "time"

// Model a basic GoLang struct which includes the following fields: ID, CreatedAt, UpdatedAt, DeletedAt
// It may be embedded into your model or you may build your own model without it
//    type User struct {
//      gorm.Model
//    }
type Model struct {
	ID        int64      `gorm:"primarykey" json:"id"`
	CreatedBy int64      `gorm:"created_by" json:"created_by,omitempty"`
	UpdatedBy int64      `gorm:"updated_by" json:"updated_by,omitempty"`
	DeletedBy int64      `gorm:"deleted_by" json:"deleted_by,omitempty"`
	Deleted   bool       `gorm:"deleted" json:"deleted"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}
