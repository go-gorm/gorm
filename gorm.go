package gorm

import "github.com/jinzhu/gorm/logger"

// Config GORM config
type Config struct {
	// SingularTable use singular table name, by default, GORM will pluralize your struct's name as table name
	// Refer https://github.com/jinzhu/inflection for inflection rules
	SingularTable bool

	// BlockGlobalUpdate generates an error on update/delete without where clause, this is to prevent eventual error with empty objects updates/deletions
	BlockGlobalUpdate bool

	Logger  logger.Interface
	LogMode logger.LogMode
}
