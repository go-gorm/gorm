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
	// DryRun generate sql without execute
	DryRun bool

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
	clone        int
}

// Session session config when create session with Session() method
type Session struct {
	DryRun         bool
	WithConditions bool
	Context        context.Context
	Logger         logger.Interface
	NowFunc        func() time.Time
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

	db = &DB{Config: config, clone: 1}

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
		txConfig = *db.Config
		tx       = &DB{
			Config:    &txConfig,
			Statement: db.Statement,
			clone:     1,
		}
	)

	if config.Context != nil {
		if tx.Statement != nil {
			tx.Statement = tx.Statement.clone()
			tx.Statement.DB = tx
		} else {
			tx.Statement = &Statement{
				DB:       tx,
				Clauses:  map[string]clause.Clause{},
				ConnPool: tx.ConnPool,
			}
		}

		tx.Statement.Context = config.Context
	}

	if config.WithConditions {
		tx.clone = 3
	}

	if config.DryRun {
		tx.Config.DryRun = true
	}

	if config.Logger != nil {
		tx.Config.Logger = config.Logger
	}

	if config.NowFunc != nil {
		tx.Config.NowFunc = config.NowFunc
	}

	return tx
}

// WithContext change current instance db's context to ctx
func (db *DB) WithContext(ctx context.Context) *DB {
	return db.Session(&Session{WithConditions: true, Context: ctx})
}

// Debug start debug mode
func (db *DB) Debug() (tx *DB) {
	return db.Session(&Session{
		WithConditions: true,
		Logger:         db.Logger.LogMode(logger.Info),
	})
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

// InstanceSet store value with key into current db instance's context
func (db *DB) InstanceSet(key string, value interface{}) *DB {
	tx := db.getInstance()
	tx.Statement.Settings.Store(fmt.Sprintf("%p", tx.Statement)+key, value)
	return tx
}

// InstanceGet get value with key from current db instance's context
func (db *DB) InstanceGet(key string) (interface{}, bool) {
	if db.Statement != nil {
		return db.Statement.Settings.Load(fmt.Sprintf("%p", db.Statement) + key)
	}
	return nil, false
}

func (db *DB) SetupJoinTable(model interface{}, field string, joinTable interface{}) error {
	var (
		tx                      = db.getInstance()
		stmt                    = tx.Statement
		modelSchema, joinSchema *schema.Schema
	)

	if err := stmt.Parse(model); err == nil {
		modelSchema = stmt.Schema
	} else {
		return err
	}

	if err := stmt.Parse(joinTable); err == nil {
		joinSchema = stmt.Schema
	} else {
		return err
	}

	if relation, ok := modelSchema.Relationships.Relations[field]; ok && relation.JoinTable != nil {
		for _, ref := range relation.References {
			if f := joinSchema.LookUpField(ref.ForeignKey.DBName); f != nil {
				ref.ForeignKey = f
			} else {
				return fmt.Errorf("missing field %v for join table", ref.ForeignKey.DBName)
			}
		}

		relation.JoinTable = joinSchema
	} else {
		return fmt.Errorf("failed to found relation: %v", field)
	}

	return nil
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
	if db.clone > 0 {
		tx := &DB{Config: db.Config}

		switch db.clone {
		case 1: // clone with new statement
		case 2: // with old statement, generate new statement for future call, used to pass to callbacks
			db.clone = 1
			tx.Statement = db.Statement
		case 3: // with clone statement
			if db.Statement != nil {
				tx.Statement = db.Statement.clone()
				tx.Statement.DB = tx
			}
		}

		if tx.Statement == nil {
			tx.Statement = &Statement{
				DB:      tx,
				Clauses: map[string]clause.Clause{},
			}
		}

		if db.Statement != nil {
			tx.Statement.Context = db.Statement.Context
			tx.Statement.ConnPool = db.Statement.ConnPool
		} else {
			tx.Statement.Context = context.Background()
			tx.Statement.ConnPool = db.ConnPool
		}

		return tx
	}

	return db
}

func Expr(expr string, args ...interface{}) clause.Expr {
	return clause.Expr{SQL: expr, Vars: args}
}
