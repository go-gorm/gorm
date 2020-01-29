package gorm

import (
	"context"
	"time"

	"github.com/jinzhu/gorm/clause"
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
	BindVar(stmt Statement, v interface{}) string
}

// Result
type Result struct {
	Error        error
	RowsAffected int64
	Statement    *Statement
}

// DB GORM DB definition
type DB struct {
	*Config
	Dialector
	Result
	Context context.Context
}

// WithContext change current instance db's context to ctx
func (db *DB) WithContext(ctx context.Context) *DB {
	tx := db.getInstance()
	tx.Context = ctx
	return tx
}

// Set store value with key into current db instance's context
func (db *DB) Set(key string, value interface{}) *DB {
	tx := db.getInstance()
	tx.Statement.Settings.Store(key, value)
	return tx
}

// Get get value with key from current db instance's context
func (db *DB) Get(key string) (interface{}, bool) {
	if db.Statement != nil {
		return db.Statement.Settings.Load(key)
	}
	return nil, false
}

func (db *DB) Close() *DB {
	// TODO
	return db
}

func (db *DB) getInstance() *DB {
	// db.Result.Statement == nil means root DB
	if db.Result.Statement == nil {
		return &DB{
			Config:    db.Config,
			Dialector: db.Dialector,
			Context:   context.Background(),
			Result: Result{
				Statement: &Statement{DB: db, Clauses: map[string][]clause.Condition{}},
			},
		}
	}

	return db
}

// Debug start debug mode
func (db *DB) Debug() (tx *DB) {
	tx = db.getInstance()
	return
}

// Session start session mode
func (db *DB) Session() (tx *DB) {
	tx = db.getInstance()
	return
}
