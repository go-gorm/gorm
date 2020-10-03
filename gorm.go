package gorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Config GORM config
type Config struct {
	// GORM perform single create, update, delete operations in transactions by default to ensure database data integrity
	// You can disable it by setting `SkipDefaultTransaction` to true
	SkipDefaultTransaction bool
	// NamingStrategy tables, columns naming strategy
	NamingStrategy schema.Namer
	// FullSaveAssociations full save associations
	FullSaveAssociations bool
	// Logger
	Logger logger.Interface
	// NowFunc the function to be used when creating a new timestamp
	NowFunc func() time.Time
	// DryRun generate sql without execute
	DryRun bool
	// PrepareStmt executes the given query in cached statement
	PrepareStmt bool
	// DisableAutomaticPing
	DisableAutomaticPing bool
	// DisableForeignKeyConstraintWhenMigrating
	DisableForeignKeyConstraintWhenMigrating bool
	// AllowGlobalUpdate allow global update
	AllowGlobalUpdate bool

	// ClauseBuilders clause builder
	ClauseBuilders map[string]clause.ClauseBuilder
	// ConnPool db conn pool
	ConnPool ConnPool
	// Dialector database dialector
	Dialector
	// Plugins registered plugins
	Plugins map[string]Plugin

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
	DryRun                 bool
	PrepareStmt            bool
	WithConditions         bool
	SkipDefaultTransaction bool
	AllowGlobalUpdate      bool
	FullSaveAssociations   bool
	Context                context.Context
	Logger                 logger.Interface
	NowFunc                func() time.Time
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

	if config.Plugins == nil {
		config.Plugins = map[string]Plugin{}
	}

	if config.cacheStore == nil {
		config.cacheStore = &sync.Map{}
	}

	db = &DB{Config: config, clone: 1}

	db.callbacks = initializeCallbacks(db)

	if config.ClauseBuilders == nil {
		config.ClauseBuilders = map[string]clause.ClauseBuilder{}
	}

	if config.Dialector != nil {
		err = config.Dialector.Initialize(db)
	}

	preparedStmt := &PreparedStmtDB{
		ConnPool:    db.ConnPool,
		Stmts:       map[string]*sql.Stmt{},
		PreparedSQL: make([]string, 0, 100),
	}
	db.cacheStore.Store("preparedStmt", preparedStmt)

	if config.PrepareStmt {
		db.ConnPool = preparedStmt
	}

	db.Statement = &Statement{
		DB:       db,
		ConnPool: db.ConnPool,
		Context:  context.Background(),
		Clauses:  map[string]clause.Clause{},
	}

	if err == nil && !config.DisableAutomaticPing {
		if pinger, ok := db.ConnPool.(interface{ Ping() error }); ok {
			err = pinger.Ping()
		}
	}

	if err != nil {
		config.Logger.Error(context.Background(), "failed to initialize database, got error %v", err)
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

	if config.SkipDefaultTransaction {
		tx.Config.SkipDefaultTransaction = true
	}

	if config.AllowGlobalUpdate {
		txConfig.AllowGlobalUpdate = true
	}

	if config.FullSaveAssociations {
		txConfig.FullSaveAssociations = true
	}

	if config.Context != nil {
		tx.Statement = tx.Statement.clone()
		tx.Statement.DB = tx
		tx.Statement.Context = config.Context
	}

	if config.PrepareStmt {
		if v, ok := db.cacheStore.Load("preparedStmt"); ok {
			tx.Statement = tx.Statement.clone()
			preparedStmt := v.(*PreparedStmtDB)
			tx.Statement.ConnPool = &PreparedStmtDB{
				ConnPool: db.Config.ConnPool,
				Mux:      preparedStmt.Mux,
				Stmts:    preparedStmt.Stmts,
			}
			txConfig.ConnPool = tx.Statement.ConnPool
			txConfig.PrepareStmt = true
		}
	}

	if config.WithConditions {
		tx.clone = 2
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
	return db.Statement.Settings.Load(key)
}

// InstanceSet store value with key into current db instance's context
func (db *DB) InstanceSet(key string, value interface{}) *DB {
	tx := db.getInstance()
	tx.Statement.Settings.Store(fmt.Sprintf("%p", tx.Statement)+key, value)
	return tx
}

// InstanceGet get value with key from current db instance's context
func (db *DB) InstanceGet(key string) (interface{}, bool) {
	return db.Statement.Settings.Load(fmt.Sprintf("%p", db.Statement) + key)
}

// Callback returns callback manager
func (db *DB) Callback() *callbacks {
	return db.callbacks
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

// DB returns `*sql.DB`
func (db *DB) DB() (*sql.DB, error) {
	connPool := db.ConnPool

	if stmtDB, ok := connPool.(*PreparedStmtDB); ok {
		connPool = stmtDB.ConnPool
	}

	if sqldb, ok := connPool.(*sql.DB); ok {
		return sqldb, nil
	}

	return nil, errors.New("invalid db")
}

func (db *DB) getInstance() *DB {
	if db.clone > 0 {
		tx := &DB{Config: db.Config}

		if db.clone == 1 {
			// clone with new statement
			tx.Statement = &Statement{
				DB:       tx,
				ConnPool: db.Statement.ConnPool,
				Context:  db.Statement.Context,
				Clauses:  map[string]clause.Clause{},
			}
		} else {
			// with clone statement
			tx.Statement = db.Statement.clone()
			tx.Statement.DB = tx
		}

		return tx
	}

	return db
}

func Expr(expr string, args ...interface{}) clause.Expr {
	return clause.Expr{SQL: expr, Vars: args}
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
				f.DataType = ref.ForeignKey.DataType
				f.GORMDataType = ref.ForeignKey.GORMDataType
				if f.Size == 0 {
					f.Size = ref.ForeignKey.Size
				}
				ref.ForeignKey = f
			} else {
				return fmt.Errorf("missing field %v for join table", ref.ForeignKey.DBName)
			}
		}

		for name, rel := range relation.JoinTable.Relationships.Relations {
			if _, ok := joinSchema.Relationships.Relations[name]; !ok {
				rel.Schema = joinSchema
				joinSchema.Relationships.Relations[name] = rel
			}
		}

		relation.JoinTable = joinSchema
	} else {
		return fmt.Errorf("failed to found relation: %v", field)
	}

	return nil
}

func (db *DB) Use(plugin Plugin) (err error) {
	name := plugin.Name()
	if _, ok := db.Plugins[name]; !ok {
		if err = plugin.Initialize(db); err == nil {
			db.Plugins[name] = plugin
		}
	} else {
		return ErrRegistered
	}

	return err
}
