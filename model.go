package gorm

import "time"

// Model is an optional embedded model providing a uint primary key, automatic
// creation/update timestamps, and indexed soft deletion through DeletedAt.
// Define your own fields instead when different key types or lifecycle behavior
// are required. Embedding Model makes ordinary Delete soft-delete the record;
// Unscoped is required for physical deletion.
//
//	type User struct {
//		gorm.Model
//		Name string
//	}
//
// See https://gorm.io/docs/models.html.
type Model struct {
	// ID is the primary key; a zero value normally lets the database generate it.
	ID uint `gorm:"primarykey"`
	// CreatedAt is populated on creation when not already set.
	CreatedAt time.Time
	// UpdatedAt is maintained by normal create/update callbacks; UpdateColumn skips
	// automatic update-time handling.
	UpdatedAt time.Time
	// DeletedAt marks soft deletion. An invalid value represents an active row; normal
	// queries filter deleted rows.
	DeletedAt DeletedAt `gorm:"index"`
}
