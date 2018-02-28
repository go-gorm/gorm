package gorm

import (
	"time"

	"github.com/jinzhu/gorm/logger"
)

var one int64 = 1

// Config GORM config
type Config struct {
	// SingularTable use singular table name, by default, GORM will pluralize your struct's name as table name
	// Refer https://github.com/jinzhu/inflection for inflection rules
	SingularTable bool

	// BlockGlobalUpdate generates an error on update/delete without where clause, this is to prevent eventual error with empty objects updates/deletions
	BlockGlobalUpdate bool

	// Logger
	Logger  logger.Interface
	LogMode logger.LogLevel

	// Dialect DB Dialect
	Dialect Dialect
}

// DB GORM DB definition
type DB struct {
	TxDialect Dialect
	Statement *Statement

	// Global config
	Config *Config

	// Result result fields
	Error        error
	RowsAffected int64
}

// Model base model definition, including fields `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`, which can be embedded in your model
//    type User struct {
//      gorm.Model
//    }
type Model struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}
