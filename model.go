package gorm

import "time"

// Model a basic GoLang struct which includes the following fields: ID, CreatedAt, UpdatedAt, DeletedAt
// It may be embedded into your model or you may build your own model without it
//    type User struct {
//      gorm.Model
//    }
type Model struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt DeletedAt `gorm:"index"`
}

// UnDelete clear model's deleted info, you need save this model manually
func (m *Model) UnDelete() {
	m.DeletedAt.Valid = false
}

// ModelSupportUnique model that support soft delete and unique index
//
// For example
//
// when you annouce an unique index for column A,
// gorm will automate create a composite index for A & deleted_flag.
//
// create: deleted_flag default to 0
// delete: set deleted_flag's value to primary id to avoid unique conflict
type ModelSupportUnique struct {
	Model
	DeletedFlag DeletedFlag `gorm:"type:BIGINT UNSIGNED NOT NULL DEFAULT 0" json:"deleted_flag"`
}

// UnDelete clear model's deleted info, you need save this model manually
func (m *ModelSupportUnique) UnDelete() {
	m.Model.UnDelete()
	m.DeletedFlag = 0
}
