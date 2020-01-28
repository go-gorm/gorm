package gorm

import (
	"time"

	"github.com/jinzhu/gorm/logger"
)

// Config GORM config
type Config struct {
	// Set true to use singular table name, by default, GORM will pluralize your struct's name as table name
	// Refer https://github.com/jinzhu/inflection for inflection rules
	SingularTable bool

	// GORM perform single create, update, delete operations in transactions by default to ensure database data integrity
	// You can cancel it by setting `SkipDefaultTransaction` to true
	SkipDefaultTransaction bool

	// Logger
	Logger logger.Interface

	// NowFunc the function to be used when creating a new timestamp
	NowFunc func() time.Time
}

// Model a basic GoLang struct which includes the following fields: ID, CreatedAt, UpdatedAt, DeletedAt
// It may be embeded into your model or you may build your own model without it
//    type User struct {
//      gorm.Model
//    }
type Model struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// Dialector GORM database dialector
type Dialector interface {
	Migrator() Migrator
}

// DB GORM DB definition
type DB struct {
	*Config
}
