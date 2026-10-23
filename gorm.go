package gorm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// preparedStmtDBKey identifies the shared prepared-statement manager in cacheStore.
// Session-level PrepareStmt wrappers reuse this manager's store and lock.
const preparedStmtDBKey = "preparedStmt"

// Config controls initialization and the default behavior of sessions created by Open.
// Configure it before opening the database; sessions shallow-copy it, so referenced
// maps, plugins, naming strategies, and connection pools can remain shared.
// Use DB.Session for per-operation options rather than changing a live Config.
//
// See https://gorm.io/docs/gorm_config.html.
type Config struct {
	// SkipDefaultTransaction disables the transaction wrapping a create, update, or delete
	// callback chain. Multi-statement writes, including associations, may then partially
	// succeed.
	SkipDefaultTransaction bool
	// DefaultTransactionTimeout sets a fallback timeout for Begin when the current context
	// has no deadline. A non-positive duration adds no timeout.
	DefaultTransactionTimeout time.Duration
	// DefaultContextTimeout sets a fallback timeout during callback execution when the
	// statement context has no deadline. A non-positive duration adds no timeout.
	DefaultContextTimeout time.Duration

	// NamingStrategy maps Go model and field names to database identifiers. Set it before
	// schemas are cached; the default is schema.NamingStrategy.
	NamingStrategy schema.Namer
	// FullSaveAssociations updates associated records in full when saving a model, rather
	// than only maintaining association references. Select and Omit still control included
	// associations.
	FullSaveAssociations bool
	// Logger receives operation traces and diagnostics. Open uses logger.Default when it is
	// nil.
	Logger logger.Interface
	// NowFunc supplies timestamps for automatic create/update times and soft deletion. Open
	// defaults to time.Now().Local; return the desired location when overriding it.
	NowFunc func() time.Time
	// DryRun builds SQL and bound variables without normal database execution. Callbacks
	// still run; inspect the returned Statement and Error rather than expecting fetched
	// rows.
	DryRun bool
	// PrepareStmt wraps the connection pool with a shared prepared-statement cache. SQL
	// text is the cache key; bound values do not create distinct entries.
	PrepareStmt bool
	// PrepareStmtMaxSize limits cached prepared statements. A non-positive value uses the
	// statement store default (effectively unbounded capacity).
	PrepareStmtMaxSize int
	// PrepareStmtTTL controls prepared-statement expiration. A non-positive value
	// uses the statement store default of 24 hours.
	PrepareStmtTTL time.Duration

	// DisableAutomaticPing skips the ping performed by Open. Initialization success then
	// does not prove the database is reachable.
	DisableAutomaticPing bool
	// DisableForeignKeyConstraintWhenMigrating prevents AutoMigrate from creating
	// foreign-key constraints. It does not disable association operations.
	DisableForeignKeyConstraintWhenMigrating bool
	// IgnoreRelationshipsWhenMigrating skips relationship traversal and relationship
	// constraints during migration.
	IgnoreRelationshipsWhenMigrating bool
	// DisableNestedTransaction makes nested Transaction callbacks use the current
	// transaction without savepoints. An inner error alone cannot undo only the inner
	// writes.
	DisableNestedTransaction bool
	// AllowGlobalUpdate permits UPDATE and DELETE without a user WHERE condition. Keep it
	// false unless an operation intentionally affects every matching row.
	AllowGlobalUpdate bool
	// QueryFields selects model columns explicitly instead of using a wildcard when
	// building queries.
	QueryFields bool
	// CreateBatchSize sets the default maximum number of records per INSERT batch when
	// positive. Explicit CreateInBatches calls supply their own size.
	CreateBatchSize int
	// TranslateError asks a dialector implementing ErrorTranslator to map driver errors to
	// GORM errors. Translation availability and coverage depend on the driver.
	TranslateError bool
	// PropagateUnscoped carries Unscoped into fresh nested statements, including
	// association operations. This can extend permanent-deletion behavior to related
	// records.
	PropagateUnscoped bool

	// ClauseBuilders overrides SQL generation by clause name. Dialectors use this shared
	// map for database-specific syntax.
	ClauseBuilders map[string]clause.ClauseBuilder
	// ConnPool is the underlying execution pool or connection wrapper. The current
	// Statement.ConnPool can instead hold a transaction or dedicated connection.
	ConnPool ConnPool
	// Dialector database dialector
	Dialector
	// Plugins contains registered plugins by name. Configure through Use before concurrent
	// use; sessions may share the map.
	Plugins map[string]Plugin

	// callbacks holds the operation callback registry shared by sessions.
	callbacks *callbacks
	// cacheStore shares parsed schemas and the prepared-statement manager across sessions.
	cacheStore *sync.Map
}

// Apply shallow-copies c into config unless both pointers are identical.
// It implements Option; maps, pointers, and the connection pool are not deep-copied.
func (c *Config) Apply(config *Config) error {
	if config != c {
		*config = *c
	}
	return nil
}

// AfterInitialize initializes the plugins in c.Plugins after Open initializes db.
// It returns the first plugin error. Map iteration does not define plugin order;
// plugins with ordering requirements should be registered explicitly with DB.Use.
func (c *Config) AfterInitialize(db *DB) error {
	if db != nil {
		for _, plugin := range c.Plugins {
			if err := plugin.Initialize(db); err != nil {
				return err
			}
		}
	}
	return nil
}

// Option participates in Open's configuration and post-initialization phases.
// Apply runs before the dialector initializes the connection. AfterInitialize runs
// on return from Open, except when dialector initialization fails; implementations
// must account for a nil DB or an initialization error. A *Config is applied first.
//
// AfterInitialize calls are deferred in reverse order of successful Apply calls. A
// later post-initialization error can replace an earlier Open error; implementations
// should not infer successful initialization merely from receiving a non-nil DB.
type Option interface {
	// Apply updates the configuration before dialector initialization. Return an error
	// to stop Open; callers must not assume a DB has been created at this stage.
	Apply(*Config) error
	// AfterInitialize performs post-initialization setup when Open returns.
	// Account for a nil DB or failed setup; see Option for the initialization order.
	AfterInitialize(*DB) error
}

// DB is a database handle, a statement builder, and the result of a traditional
// API operation. Open returns a reusable root handle backed by a connection pool;
// it is not a dedicated connection and does not itself denote a transaction.
//
// Chain methods such as Where return a handle carrying mutable statement state.
// Do not share an initialized chain across goroutines or independent queries.
// Use Session(&Session{}) to derive a reusable base that preserves conditions,
// or Session(&Session{NewDB: true}) to start without those conditions.
//
// Finishers such as Find and Create execute callbacks and return a DB whose Error
// and RowsAffected describe the operation. Always check Error before using results.
// The generic G API instead accepts context at execution and returns errors directly.
type DB struct {
	*Config
	// Error records errors from the current chain; inspect it on the returned finisher
	// result.
	Error error
	// RowsAffected records rows reported by the operation. Its meaning depends on the
	// operation and driver; check Error first.
	RowsAffected int64
	// Statement contains mutable query state, bound variables, context, and the active
	// connection. Use chain methods instead of sharing or mutating it concurrently.
	Statement *Statement
	// clone selects fresh-statement, copied-statement, or in-place behavior in getInstance.
	clone int
}

// Session specifies per-session changes for DB.Session.
// The zero value preserves existing configuration and query conditions while
// making subsequent chains independent. Most boolean options only enable a
// behavior; false does not clear a value already enabled in Config.
// NewDB discards conditions on the next operation, not the connection or transaction.
//
// See https://gorm.io/docs/session.html.
type Session struct {
	// DryRun builds SQL and bound variables without normal database execution. Callbacks
	// still run; inspect the returned Statement and Error rather than expecting fetched
	// rows.
	// It applies to the session returned by DB.Session.
	DryRun bool
	// PrepareStmt enables prepared-statement caching for the returned session.
	// Sessions can share the same store, so clearing it affects other users of that store.
	PrepareStmt bool
	// NewDB starts subsequent operations with an empty statement rather than copying
	// conditions. The underlying pool or transaction and context remain in use.
	NewDB bool
	// Initialized eagerly initializes the new statement. The resulting handle is not safe
	// for independent chain reuse or concurrent mutation.
	Initialized bool
	// SkipHooks skips model lifecycle hooks for this session. It does not remove registered
	// callbacks or turn off association handling.
	SkipHooks bool
	// SkipDefaultTransaction disables the transaction wrapping a create, update, or delete
	// callback chain. Multi-statement writes, including associations, may then partially
	// succeed.
	// It applies to the session returned by DB.Session.
	SkipDefaultTransaction bool
	// DisableNestedTransaction makes nested Transaction callbacks use the current
	// transaction without savepoints. An inner error alone cannot undo only the inner
	// writes.
	// It applies to the session returned by DB.Session.
	DisableNestedTransaction bool
	// AllowGlobalUpdate permits UPDATE and DELETE without a user WHERE condition. Keep it
	// false unless an operation intentionally affects every matching row.
	// It applies to the session returned by DB.Session.
	AllowGlobalUpdate bool
	// FullSaveAssociations updates associated records in full when saving a model, rather
	// than only maintaining association references. Select and Omit still control included
	// associations.
	// It applies to the session returned by DB.Session.
	FullSaveAssociations bool
	// PropagateUnscoped carries Unscoped into fresh nested statements, including
	// association operations. This can extend permanent-deletion behavior to related
	// records.
	// It applies to the session returned by DB.Session.
	PropagateUnscoped bool
	// QueryFields selects model columns explicitly instead of using a wildcard when
	// building queries.
	// It applies to the session returned by DB.Session.
	QueryFields bool
	// Context overrides the context used by subsequent SQL operations, callbacks, and
	// logging when non-nil.
	Context context.Context
	// Logger overrides the session logger when non-nil; nil preserves the existing logger.
	Logger logger.Interface
	// NowFunc overrides the timestamp function when non-nil; nil preserves the
	// existing function. Return times in the location required by the application.
	NowFunc func() time.Time
	// CreateBatchSize overrides the default insert batch size when positive.
	// Zero or a negative value preserves the existing setting.
	CreateBatchSize int
}

// Open initializes a reusable database handle using a database-specific dialector
// and configuration options. Import a driver such as gorm.io/driver/sqlite and pass
// its Open result as dialector; this package does not select a driver from a DSN.
//
// Open installs callbacks, applies options, initializes the dialector and plugins,
// and pings the connection when supported unless DisableAutomaticPing is set.
// Always check err even when db is non-nil. To configure pool limits or close the
// underlying pool at application shutdown, obtain it with DB.DB.
//
// See https://gorm.io/docs/connecting_to_the_database.html.
func Open(dialector Dialector, opts ...Option) (db *DB, err error) {
	config := &Config{}

	sort.Slice(opts, func(i, j int) bool {
		_, isConfig := opts[i].(*Config)
		_, isConfig2 := opts[j].(*Config)
		return isConfig && !isConfig2
	})

	if len(opts) > 0 {
		if c, ok := opts[0].(*Config); ok {
			config = c
		} else {
			opts = append([]Option{config}, opts...)
		}
	}

	var skipAfterInitialize bool
	for _, opt := range opts {
		if opt != nil {
			if applyErr := opt.Apply(config); applyErr != nil {
				return nil, applyErr
			}
			defer func(opt Option) {
				if skipAfterInitialize {
					return
				}
				if errr := opt.AfterInitialize(db); errr != nil {
					err = errr
				}
			}(opt)
		}
	}

	if d, ok := dialector.(interface{ Apply(*Config) error }); ok {
		if err = d.Apply(config); err != nil {
			return
		}
	}

	if config.NamingStrategy == nil {
		config.NamingStrategy = schema.NamingStrategy{IdentifierMaxLength: 64} // Default Identifier length is 64
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
		if err != nil {
			if db, _ := db.DB(); db != nil {
				_ = db.Close()
			}

			// DB is not initialized, so we skip AfterInitialize
			skipAfterInitialize = true
			return
		}

		if config.TranslateError {
			if _, ok := db.Dialector.(ErrorTranslator); !ok {
				config.Logger.Warn(context.Background(), "The TranslateError option is enabled, but the Dialector %s does not implement ErrorTranslator.", db.Dialector.Name())
			}
		}
	}

	if config.PrepareStmt {
		preparedStmt := NewPreparedStmtDB(db.ConnPool, config.PrepareStmtMaxSize, config.PrepareStmtTTL)
		db.cacheStore.Store(preparedStmtDBKey, preparedStmt)
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
			if err != nil {
				if db, _ := db.DB(); db != nil {
					_ = db.Close()
				}
			}
		}
	}

	if err != nil {
		config.Logger.Error(context.Background(), "failed to initialize database, got error %v", err)
	}

	return
}

// Session returns a handle with a shallow copy of the configuration and the
// requested overrides. Session(&Session{}) preserves conditions but clones statement
// state for subsequent chains; NewDB starts subsequent chains without conditions.
// The connection pool, including an active transaction, remains shared.
//
// Initialized eagerly initializes the statement and removes this safe-reuse
// boundary: do not share that result across independent chains or goroutines.
// Session does not clear an existing Error and does not begin a transaction.
func (db *DB) Session(config *Session) *DB {
	var (
		txConfig = *db.Config
		tx       = &DB{
			Config:    &txConfig,
			Statement: db.Statement,
			Error:     db.Error,
			clone:     1,
		}
	)
	if config.CreateBatchSize > 0 {
		tx.Config.CreateBatchSize = config.CreateBatchSize
	}

	if config.SkipDefaultTransaction {
		tx.Config.SkipDefaultTransaction = true
	}

	if config.AllowGlobalUpdate {
		txConfig.AllowGlobalUpdate = true
	}

	if config.FullSaveAssociations {
		txConfig.FullSaveAssociations = true
	}

	if config.PropagateUnscoped {
		txConfig.PropagateUnscoped = true
	}

	if config.Context != nil || config.PrepareStmt || config.SkipHooks {
		tx.Statement = tx.Statement.clone()
		tx.Statement.DB = tx
	}

	if config.Context != nil {
		tx.Statement.Context = config.Context
	}

	if config.PrepareStmt {
		var preparedStmt *PreparedStmtDB

		if v, ok := db.cacheStore.Load(preparedStmtDBKey); ok {
			preparedStmt = v.(*PreparedStmtDB)
		} else {
			preparedStmt = NewPreparedStmtDB(db.ConnPool, db.PrepareStmtMaxSize, db.PrepareStmtTTL)
			db.cacheStore.Store(preparedStmtDBKey, preparedStmt)
		}

		switch t := tx.Statement.ConnPool.(type) {
		case Tx:
			tx.Statement.ConnPool = &PreparedStmtTX{
				Tx:             t,
				PreparedStmtDB: preparedStmt,
			}
		default:
			tx.Statement.ConnPool = &PreparedStmtDB{
				ConnPool: db.Config.ConnPool,
				Mux:      preparedStmt.Mux,
				Stmts:    preparedStmt.Stmts,
			}
		}
		txConfig.ConnPool = tx.Statement.ConnPool
		txConfig.PrepareStmt = true
	}

	if config.SkipHooks {
		tx.Statement.SkipHooks = true
	}

	if config.DisableNestedTransaction {
		txConfig.DisableNestedTransaction = true
	}

	if !config.NewDB {
		tx.clone = 2
	}

	if config.DryRun {
		tx.Config.DryRun = true
	}

	if config.QueryFields {
		tx.Config.QueryFields = true
	}

	if config.Logger != nil {
		tx.Config.Logger = config.Logger
	}

	if config.NowFunc != nil {
		tx.Config.NowFunc = config.NowFunc
	}

	if config.Initialized {
		tx = tx.getInstance()
	}

	return tx
}

// WithContext returns a new session using ctx for SQL execution, hooks, and logging.
// It is shorthand for Session(&Session{Context: ctx}) and preserves conditions.
// Pass a non-nil context; cancellation is propagated to the underlying driver.
// Call the cancel function yourself when creating a timeout context.
func (db *DB) WithContext(ctx context.Context) *DB {
	return db.Session(&Session{Context: ctx})
}

// Debug returns a new session whose logger runs at Info level, so subsequent SQL
// operations can be traced. It preserves conditions and does not execute a query.
// Log output may contain parameter values unless the logger filters them.
func (db *DB) Debug() (tx *DB) {
	tx = db.getInstance()
	return tx.Session(&Session{
		Logger: db.Logger.LogMode(logger.Info),
	})
}

// Set stores a setting on the returned statement for callbacks and hooks to read
// with Get. This is Statement.Settings, not context.Context.WithValue.
// Settings are copied when a statement is cloned and may reach association hooks;
// use InstanceSet when a value must belong only to this statement instance.
func (db *DB) Set(key string, value interface{}) *DB {
	tx := db.getInstance()
	tx.Statement.Settings.Store(key, value)
	return tx
}

// Get looks up a setting written with Set on the current statement.
// The boolean reports whether the key exists, including a stored nil value.
// It does not read values from Statement.Context.
func (db *DB) Get(key string) (interface{}, bool) {
	return db.Statement.Settings.Load(key)
}

// InstanceSet stores a setting under a key scoped to the returned Statement's
// identity. InstanceGet on that same statement retrieves it; a newly created
// association statement does not inherit access to this instance-specific key.
func (db *DB) InstanceSet(key string, value interface{}) *DB {
	tx := db.getInstance()
	tx.Statement.Settings.Store(fmt.Sprintf("%p", tx.Statement)+key, value)
	return tx
}

// InstanceGet retrieves a value set with InstanceSet for this exact statement.
// It returns false for a missing key or for a key belonging to another statement,
// even if that other statement's settings were copied into this one.
func (db *DB) InstanceGet(key string) (interface{}, bool) {
	return db.Statement.Settings.Load(fmt.Sprintf("%p", db.Statement) + key)
}

// Callback returns the shared callback registry used by finishers.
// Plugins register handlers on its Create, Query, Update, Delete, Row, or Raw
// processors. Registration affects sessions sharing this registry; configure it
// before concurrent database use. See https://gorm.io/docs/write_plugins.html.
func (db *DB) Callback() *callbacks {
	return db.callbacks
}

// AddError records err on db and returns the accumulated Error; nil leaves it
// unchanged.
// With TranslateError enabled it first invokes the dialector's ErrorTranslator.
// When an error already exists, its text is retained and the new error is wrapped
// with %w; errors.Is does not necessarily match earlier accumulated errors.
// An attached WithResult collector is updated with the resulting error.
func (db *DB) AddError(err error) error {
	if err != nil {
		if db.Config.TranslateError {
			if errTranslator, ok := db.Dialector.(ErrorTranslator); ok {
				err = errTranslator.Translate(err)
			}
		}

		if db.Error == nil {
			db.Error = err
		} else {
			db.Error = fmt.Errorf("%v; %w", db.Error, err)
		}
		if db.Statement != nil && db.Statement.Result != nil {
			db.Statement.Result.Error = db.Error
		}
	}
	return db.Error
}

// DB exposes the underlying *sql.DB for pool configuration, statistics, or shutdown.
// It examines the current statement's connection first, supports *sql.Tx and
// GetDBConnector wrappers, and returns ErrInvalidDB when no *sql.DB is available
// (for example, a dedicated *sql.Conn without a connector).
// The returned pool is shared; closing it affects other sessions. Using this pool
// directly does not execute inside the current GORM transaction.
func (db *DB) DB() (*sql.DB, error) {
	connPool := db.ConnPool
	if db.Statement != nil && db.Statement.ConnPool != nil {
		connPool = db.Statement.ConnPool
	}
	if tx, ok := connPool.(*sql.Tx); ok && tx != nil {
		return (*sql.DB)(reflect.ValueOf(tx).Elem().FieldByName("db").UnsafePointer()), nil
	}

	if dbConnector, ok := connPool.(GetDBConnector); ok && dbConnector != nil {
		if sqldb, err := dbConnector.GetDBConn(); sqldb != nil || err != nil {
			return sqldb, err
		}
	}

	if sqldb, ok := connPool.(*sql.DB); ok && sqldb != nil {
		return sqldb, nil
	}

	return nil, ErrInvalidDB
}

// getInstance implements lazy statement isolation for chaining.
// clone=1 creates a fresh statement carrying connection, context, and hook settings;
// clone=2 copies the existing statement. An initialized handle (clone=0) is returned
// unchanged, so callers must not treat every chain method as an immutable operation.
func (db *DB) getInstance() *DB {
	if db.clone > 0 {
		tx := &DB{Config: db.Config, Error: db.Error}

		if db.clone == 1 {
			// clone with new statement
			tx.Statement = &Statement{
				DB:        tx,
				ConnPool:  db.Statement.ConnPool,
				Context:   db.Statement.Context,
				Clauses:   map[string]clause.Clause{},
				Vars:      make([]interface{}, 0, 8),
				SkipHooks: db.Statement.SkipHooks,
			}
			if db.Config.PropagateUnscoped {
				tx.Statement.Unscoped = db.Statement.Unscoped
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

// Expr builds a SQL expression for use as a value or clause, for example
// DB.Model(&account).Update("balance", Expr("balance + ?", delta)).
// The SQL template is trusted SQL; args are bound through the dialect's placeholders.
// Do not interpolate untrusted text into expr, including identifiers or operators.
// Expr only constructs an expression and does not execute it.
func Expr(expr string, args ...interface{}) clause.Expr {
	return clause.Expr{SQL: expr, Vars: args}
}

// SetupJoinTable replaces the many-to-many join-table schema for model's named
// association field. joinTable must contain the foreign-key columns required by
// the relation; missing relations or columns return an error.
// Register the custom model before using the association or migrating it. This
// changes cached schema metadata and does not create or alter database tables.
// See https://gorm.io/docs/many_to_many.html#Customize-JoinTable.
func (db *DB) SetupJoinTable(model interface{}, field string, joinTable interface{}) error {
	var (
		tx                      = db.getInstance()
		stmt                    = tx.Statement
		modelSchema, joinSchema *schema.Schema
	)

	err := stmt.Parse(model)
	if err != nil {
		return err
	}
	modelSchema = stmt.Schema

	err = stmt.Parse(joinTable)
	if err != nil {
		return err
	}
	joinSchema = stmt.Schema

	relation, ok := modelSchema.Relationships.Relations[field]
	isRelation := ok && relation.JoinTable != nil
	if !isRelation {
		return fmt.Errorf("failed to find relation: %s", field)
	}

	for _, ref := range relation.References {
		f := joinSchema.LookUpField(ref.ForeignKey.DBName)
		if f == nil {
			return fmt.Errorf("missing field %s for join table", ref.ForeignKey.DBName)
		}

		f.DataType = ref.ForeignKey.DataType
		f.GORMDataType = ref.ForeignKey.GORMDataType
		if f.Size == 0 {
			f.Size = ref.ForeignKey.Size
		}
		ref.ForeignKey = f
	}

	for name, rel := range relation.JoinTable.Relationships.Relations {
		if _, ok := joinSchema.Relationships.Relations[name]; !ok {
			rel.Schema = joinSchema
			joinSchema.Relationships.Relations[name] = rel
		}
	}
	relation.JoinTable = joinSchema

	return nil
}

// Use initializes a plugin and registers it by Plugin.Name after successful
// initialization.
// It returns ErrRegistered for a duplicate name and leaves the plugin unregistered
// if Initialize fails. Configure plugins before concurrent use of the shared DB;
// registration mutates the shared plugin map and may modify callback registries.
func (db *DB) Use(plugin Plugin) error {
	name := plugin.Name()
	if _, ok := db.Plugins[name]; ok {
		return ErrRegistered
	}
	if err := plugin.Initialize(db); err != nil {
		return err
	}
	db.Plugins[name] = plugin
	return nil
}

// ToSQL runs queryFn in a DryRun session and returns dialect-formatted SQL for
// inspection.
// queryFn must return a finished statement, for example:
//
//	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
//		return tx.Where("name = ?", "alice").Limit(10).Find(&[]User{})
//	})
//
// The result interpolates values for debugging and is not safe executable SQL.
// Use parameterized operations to execute queries. ToSQL returns no error; inspect
// Statement.SQL, Statement.Vars, and Error on a DryRun result when those are needed.
// DryRun still invokes callbacks, so queryFn and custom callbacks must avoid side
// effects.
func (db *DB) ToSQL(queryFn func(tx *DB) *DB) string {
	tx := queryFn(db.Session(&Session{DryRun: true, SkipDefaultTransaction: true}).getInstance())
	stmt := tx.Statement

	return db.Dialector.Explain(stmt.SQL.String(), stmt.Vars...)
}
