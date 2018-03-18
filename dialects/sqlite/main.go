package sqlite

import (
	"database/sql"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/logger"
	// import sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// Config database config
type Config gorm.Config

// Open initialize GORM db connection
func Open(dsn string, config Config) (*gorm.DB, error) {
	dialect, err := New(dsn)
	config.Dialect = dialect
	gormConfig := gorm.Config(config)
	gormConfig.Logger = logger.DefaultLogger

	return &gorm.DB{Config: &gormConfig}, err
}

// New initialize sqlite dialect
func New(dsn string) (*Dialect, error) {
	dbSQL, err := sql.Open("sqlite3", dsn)
	return &Dialect{DB: dbSQL}, err
}
