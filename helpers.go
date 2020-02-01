package gorm

import (
	"errors"
	"time"
)

var (
	// ErrRecordNotFound record not found error
	ErrRecordNotFound = errors.New("record not found")
	// ErrInvalidSQL invalid SQL error, happens when you passed invalid SQL
	ErrInvalidSQL = errors.New("invalid SQL")
	// ErrInvalidTransaction invalid transaction when you are trying to `Commit` or `Rollback`
	ErrInvalidTransaction = errors.New("no valid transaction")
	// ErrUnaddressable unaddressable value
	ErrUnaddressable = errors.New("using unaddressable value")
)

// Model a basic GoLang struct which includes the following fields: ID, CreatedAt, UpdatedAt, DeletedAt
// It may be embeded into your model or you may build your own model without it
//    type User struct {
//      gorm.Model
//    }
type Model struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}
