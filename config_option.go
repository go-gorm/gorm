package gorm

import (
	"sync"
	"time"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// ConfigOption use functional option for gorm Config.
type ConfigOption func(c *Config)

// WithSkipDefaultTransaction enable SkipDefaultTransaction.
func WithSkipDefaultTransaction() ConfigOption {
	return func(c *Config) {
		c.SkipDefaultTransaction = true
	}
}

// WithNameStrategy set shema namer.
func WithNameStrategy(namer schema.Namer) ConfigOption {
	return func(c *Config) {
		c.NamingStrategy = namer
	}
}

// WithFullSaveAssociations set FullSaveAssociations = true.
func WithFullSaveAssociations() ConfigOption {
	return func(c *Config) {
		c.FullSaveAssociations = true
	}
}

// WithLogger set logger.
func WithLogger(logger logger.Interface) ConfigOption {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithNowFunc set now func.
func WithNowFunc(fn func() time.Time) ConfigOption {
	return func(c *Config) {
		c.NowFunc = fn
	}
}

// WithEnableDryRun enable dry run.
func WithEnableDryRun() ConfigOption {
	return func(c *Config) {
		c.DryRun = true
	}
}

// WithPrepareStmt enable PrepareStmt.
func WithPrepareStmt() ConfigOption {
	return func(c *Config) {
		c.PrepareStmt = true
	}
}

// WithEnableAutomaticPing enable ping.
func WithEnableAutomaticPing() ConfigOption {
	return func(c *Config) {
		c.DisableAutomaticPing = true
	}
}

// WithEnableForeignKeyConstraintWhenMigrating
// ForeignKeyConstraintWhenMigrating config.
func WithEnableForeignKeyConstraintWhenMigrating() ConfigOption {
	return func(c *Config) {
		c.DisableForeignKeyConstraintWhenMigrating = true
	}
}

// WithEnableNestedTransaction enable NestedTransaction.
func WithEnableNestedTransaction() ConfigOption {
	return func(c *Config) {
		c.DisableNestedTransaction = true
	}
}

// WithAllowGlobalUpdate allow global update.
func WithAllowGlobalUpdate() ConfigOption {
	return func(c *Config) {
		c.AllowGlobalUpdate = true
	}
}

// WithEnableQueryFields open QueryFields.
func WithEnableQueryFields() ConfigOption {
	return func(c *Config) {
		c.QueryFields = true
	}
}

// WithCreateBatchSize set batch size.
func WithCreateBatchSize(size int) ConfigOption {
	return func(c *Config) {
		c.CreateBatchSize = size
	}
}

// WithClauseBuilders set clause builder.
func WithClauseBuilders(m map[string]clause.ClauseBuilder) ConfigOption {
	return func(c *Config) {
		c.ClauseBuilders = m
	}
}

// WithConnPool set conn pool.
func WithConnPool(connPool ConnPool) ConfigOption {
	return func(c *Config) {
		c.ConnPool = connPool
	}
}

// WithDialector set dialector.
func WithDialector(dialector Dialector) ConfigOption {
	return func(c *Config) {
		c.Dialector = dialector
	}
}

// WithConfigPlugins set config plugins.
func WithConfigPlugins(m map[string]Plugin) ConfigOption {
	return func(c *Config) {
		c.Plugins = m
	}
}

// WithConfigCallbacks set cb for Config entry.
func WithConfigCallbacks(cb *callbacks) ConfigOption {
	return func(c *Config) {
		c.callbacks = cb
	}
}

// WithCacheStore set cacheStore for Config entry.
func WithCacheStore(s *sync.Map) ConfigOption {
	return func(c *Config) {
		c.cacheStore = s
	}
}
