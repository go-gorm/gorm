package gorm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/logger"
	"github.com/jinzhu/gorm/schema"
)

// Config GORM config
type Config struct {
	// GORM perform single create, update, delete operations in transactions by default to ensure database data integrity
	// You can disable it by setting `SkipDefaultTransaction` to true
	SkipDefaultTransaction bool
	// NamingStrategy tables, columns naming strategy
	NamingStrategy schema.Namer
	// Logger
	Logger logger.Interface
	// NowFunc the function to be used when creating a new timestamp
	NowFunc func() time.Time

	// ClauseBuilders clause builder
	ClauseBuilders map[string]clause.ClauseBuilder
	// ConnPool db conn pool
	ConnPool ConnPool
	// Dialector database dialector
	Dialector

	callbacks  *callbacks
	cacheStore *sync.Map
}

// DB GORM DB definition
type DB struct {
	*Config
	Error        error
	RowsAffected int64
	Statement    *Statement
	clone        bool
}

// Session session config when create session with Session() method
type Session struct {
	Context context.Context
	Logger  logger.Interface
	NowFunc func() time.Time
}

// Open initialize db session based on dialector
func Open(dialector Dialector, config *Config) (db *DB, err error) {
	if config == nil {
		config = &Config{}
	}

	if config.NamingStrategy == nil {
		config.NamingStrategy = schema.NamingStrategy{}
	}

	if config.Logger == nil {
		config.Logger = logger.Default
	}

	if config.NowFunc == nil {
		config.NowFunc = func() time.Time { return time.Now().Local() }
	}

	if dialector != nil {
		config.Dialector = dialector
	}

	if config.cacheStore == nil {
		config.cacheStore = &sync.Map{}
	}

	db = &DB{
		Config: config,
		clone:  true,
	}

	db.callbacks = initializeCallbacks(db)

	if config.ClauseBuilders == nil {
		config.ClauseBuilders = map[string]clause.ClauseBuilder{}
	}

	if dialector != nil {
		err = dialector.Initialize(db)
	}
	return
}

// Session create new db session
func (db *DB) Session(config *Session) *DB {
	var (
		tx       = db.getInstance()
		stmt     = tx.Statement.clone()
		txConfig = *tx.Config
	)

	if config.Context != nil {
		stmt.Context = config.Context
	}

	if config.Logger != nil {
		txConfig.Logger = config.Logger
	}

	if config.NowFunc != nil {
		txConfig.NowFunc = config.NowFunc
	}

	return &DB{
		Config:    &txConfig,
		Statement: stmt,
		clone:     true,
	}
}

// WithContext change current instance db's context to ctx
func (db *DB) WithContext(ctx context.Context) *DB {
	return db.Session(&Session{Context: ctx})
}

// Debug start debug mode
func (db *DB) Debug() (tx *DB) {
	return db.Session(&Session{Logger: db.Logger.LogMode(logger.Info)})
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

// Callback returns callback manager
func (db *DB) Callback() *callbacks {
	return db.callbacks
}

// AutoMigrate run auto migration for given models
func (db *DB) AutoMigrate(dst ...interface{}) error {
	return db.Migrator().AutoMigrate(dst...)
}

// AddError add error to db
func (db *DB) AddError(err error) error {
	if db.Error == nil {
		db.Error = err
	} else if err != nil {
		db.Error = fmt.Errorf("%v; %w", db.Error, err)
	}
	return db.Error
}

func (db *DB) getInstance() *DB {
	if db.clone {
		stmt := &Statement{
			DB:       db,
			ConnPool: db.ConnPool,
			Clauses:  map[string]clause.Clause{},
			Context:  context.Background(),
		}

		if db.Statement != nil {
			stmt.Context = db.Statement.Context
		}

		return &DB{Config: db.Config, Statement: stmt}
	}

	return db
}

func Expr(expr string, args ...interface{}) clause.Expr {
	return clause.Expr{SQL: expr, Vars: args}
}
