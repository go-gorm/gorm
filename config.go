package gorm

import "github.com/jinzhu/gorm/logger"

// Config GORM config
type Config struct {
	// MaxIdleConnections sets the maximum number of connections in the idle connection pool
	MaxIdleConnections int
	// MaxOpenConnections sets the maximum number of open connections to the database
	MaxOpenConnections int

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

	// Callbacks defined GORM callbacks
	Callbacks *Callback

	// db fresh db connection
	globalDbConnection SQLCommon
}
