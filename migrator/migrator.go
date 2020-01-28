package migrator

import "github.com/jinzhu/gorm"

// Migrator migrator struct
type Migrator struct {
	*Config
}

// Config schema config
type Config struct {
	CheckExistsBeforeDropping bool
	DB                        *gorm.DB
}
