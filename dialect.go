package dialects

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/builder"
)

// Dialect GORM dialect interface
type Dialect interface {
	// CRUD operations
	Insert(*gorm.DB, builder.Statement) error
	Query(*gorm.DB, builder.Statement) error
	Update(*gorm.DB, builder.Statement) error
	Delete(*gorm.DB, builder.Statement) error

	// DB Driver interface
	QueryRow(*gorm.DB) error
	QueryRows(*gorm.DB) error
	Exec(*gorm.DB) error
}
