package gorm

import (
	"database/sql"
)

// ViewOption view option
type ViewOption struct {
	Replace     bool
	CheckOption string
	Query       *DB
}

type Migrator interface {
	// AutoMigrate
	AutoMigrate(dst ...interface{}) error

	// Tables
	CreateTable(dst ...interface{}) error
	DropTable(dst ...interface{}) error
	HasTable(dst ...interface{}) error
	RenameTable(oldName, newName string) error

	// Columns
	AddColumn(dst interface{}, field string) error
	DropColumn(dst interface{}, field string) error
	AlterColumn(dst interface{}, field string) error
	RenameColumn(dst interface{}, oldName, field string) error
	ColumnTypes(dst interface{}) ([]*sql.ColumnType, error)

	// Views
	CreateView(name string, option ViewOption) error
	DropView(name string) error

	// Constraints
	CreateConstraint(dst interface{}, name string) error
	DropConstraint(dst interface{}, name string) error

	// Indexes
	CreateIndex(dst interface{}, name string) error
	DropIndex(dst interface{}, name string) error
	HasIndex(dst interface{}, name string) error
	RenameIndex(dst interface{}, oldName, newName string) error
}
