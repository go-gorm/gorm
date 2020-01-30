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

// Dialector GORM database dialector
type Dialector interface {
	Migrator() Migrator
	BindVar(stmt Statement, v interface{}) string
}

// DB GORM DB definition
type DB struct {
	*Config
	Dialector
	Instance
	clone bool
}

// Session session config when create new session
type Session struct {
	Context context.Context
	Logger  logger.Interface
	NowFunc func() time.Time
}

// Open initialize db session based on dialector
func Open(dialector Dialector, config *Config) (db *DB, err error) {
	return &DB{
		Config:    config,
		Dialector: dialector,
		clone:     true,
	}, nil
}

// Session create new db session
func (db *DB) Session(config *Session) *DB {
	var (
		tx       = db.getInstance()
		txConfig = *tx.Config
	)

	if config.Context != nil {
		tx.Context = config.Context
	}

	if config.Logger != nil {
		txConfig.Logger = config.Logger
	}

	if config.NowFunc != nil {
		txConfig.NowFunc = config.NowFunc
	}

	tx.Config = &txConfig
	tx.clone = true
	return tx
}

// WithContext change current instance db's context to ctx
func (db *DB) WithContext(ctx context.Context) *DB {
	return db.Session(&Session{Context: ctx})
}

// Debug start debug mode
func (db *DB) Debug() (tx *DB) {
	return db.Session(&Session{Logger: db.Logger.LogMode(logger.Info)})
}

func (db *DB) Close() error {
	return nil
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

func (db *DB) getInstance() *DB {
	if db.clone {
		ctx := db.Instance.Context
		if ctx == nil {
			ctx = context.Background()
		}

		return &DB{
			Config:    db.Config,
			Dialector: db.Dialector,
			Instance: Instance{
				Context:   ctx,
				Statement: &Statement{DB: db, Clauses: map[string]clause.Clause{}},
			},
		}
	}

	return db
}
