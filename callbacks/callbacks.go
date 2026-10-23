package callbacks

import (
	"gorm.io/gorm"
)

var (
	// createClauses is the default INSERT clause order. Dialectors may replace it, for
	// example to add RETURNING.
	createClauses = []string{"INSERT", "VALUES", "ON CONFLICT"}
	// queryClauses is the default SELECT clause order, also installed on the row and raw
	// processors.
	queryClauses = []string{"SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR"}
	// updateClauses is the default UPDATE clause order; dialects can supply an order that
	// includes RETURNING.
	updateClauses = []string{"UPDATE", "SET", "WHERE"}
	// deleteClauses is the default physical DELETE clause order. Field-specific clauses can
	// replace deletion with an UPDATE for soft deletion.
	deleteClauses = []string{"DELETE", "FROM", "WHERE"}
)

// Config selects dialect-specific SQL clause ordering and generated-ID interpretation
// for the default processors. Pass a non-nil Config to RegisterDefaultCallbacks; that
// function fills empty clause lists in place. Configure it during initialization rather
// than mutating it while queries run.
type Config struct {
	// LastInsertIDReversed indicates that LastInsertId identifies the last generated ID in
	// a batch. Create then works backward using the primary field's AutoIncrementIncrement;
	// otherwise it assumes the first ID and works forward. This inference requires the
	// driver/database batch-ID semantics to match.
	LastInsertIDReversed bool
	// CreateClauses lists INSERT clauses in build order. Including RETURNING enables the
	// default create callback's returning path; support must agree with the dialector.
	CreateClauses []string
	// QueryClauses lists SELECT clauses in build order, shared by query, row, and raw
	// processors. An empty list selects the defaults.
	QueryClauses []string
	// UpdateClauses lists UPDATE clauses in build order. Including RETURNING enables
	// scanning returned rows into the model.
	UpdateClauses []string
	// DeleteClauses lists DELETE clauses in build order. Including RETURNING enables
	// scanning deleted rows into the destination.
	DeleteClauses []string
}

// RegisterDefaultCallbacks installs the named create, query, update, delete, row, and
// raw callbacks and their clause orders. Dialectors normally call it from Initialize,
// after choosing supported clauses in a non-nil config. Empty clause lists are filled
// in config in place.
//
// Writes normally run begin-transaction, before hooks, association work, SQL, after
// hooks, then commit-or-rollback. Belongs-to associations are saved before the owner;
// other associations follow it. Queries run SQL, preloads, then AfterFind. SkipHooks
// skips model hooks, not the processor or association callbacks. Use db.Callback()
// registration ordering to extend these chains; changing the registry affects all
// sessions sharing that configuration.
func RegisterDefaultCallbacks(db *gorm.DB, config *Config) {
	enableTransaction := func(db *gorm.DB) bool {
		return !db.SkipDefaultTransaction
	}

	if len(config.CreateClauses) == 0 {
		config.CreateClauses = createClauses
	}
	if len(config.QueryClauses) == 0 {
		config.QueryClauses = queryClauses
	}
	if len(config.DeleteClauses) == 0 {
		config.DeleteClauses = deleteClauses
	}
	if len(config.UpdateClauses) == 0 {
		config.UpdateClauses = updateClauses
	}

	createCallback := db.Callback().Create()
	createCallback.Match(enableTransaction).Register("gorm:begin_transaction", BeginTransaction)
	createCallback.Register("gorm:before_create", BeforeCreate)
	createCallback.Register("gorm:save_before_associations", SaveBeforeAssociations(true))
	createCallback.Register("gorm:create", Create(config))
	createCallback.Register("gorm:save_after_associations", SaveAfterAssociations(true))
	createCallback.Register("gorm:after_create", AfterCreate)
	createCallback.Match(enableTransaction).Register("gorm:commit_or_rollback_transaction", CommitOrRollbackTransaction)
	createCallback.Clauses = config.CreateClauses

	queryCallback := db.Callback().Query()
	queryCallback.Register("gorm:query", Query)
	queryCallback.Register("gorm:preload", Preload)
	queryCallback.Register("gorm:after_query", AfterQuery)
	queryCallback.Clauses = config.QueryClauses

	deleteCallback := db.Callback().Delete()
	deleteCallback.Match(enableTransaction).Register("gorm:begin_transaction", BeginTransaction)
	deleteCallback.Register("gorm:before_delete", BeforeDelete)
	deleteCallback.Register("gorm:delete_before_associations", DeleteBeforeAssociations)
	deleteCallback.Register("gorm:delete", Delete(config))
	deleteCallback.Register("gorm:after_delete", AfterDelete)
	deleteCallback.Match(enableTransaction).Register("gorm:commit_or_rollback_transaction", CommitOrRollbackTransaction)
	deleteCallback.Clauses = config.DeleteClauses

	updateCallback := db.Callback().Update()
	updateCallback.Match(enableTransaction).Register("gorm:begin_transaction", BeginTransaction)
	updateCallback.Register("gorm:setup_reflect_value", SetupUpdateReflectValue)
	updateCallback.Register("gorm:before_update", BeforeUpdate)
	updateCallback.Register("gorm:save_before_associations", SaveBeforeAssociations(false))
	updateCallback.Register("gorm:update", Update(config))
	updateCallback.Register("gorm:save_after_associations", SaveAfterAssociations(false))
	updateCallback.Register("gorm:after_update", AfterUpdate)
	updateCallback.Match(enableTransaction).Register("gorm:commit_or_rollback_transaction", CommitOrRollbackTransaction)
	updateCallback.Clauses = config.UpdateClauses

	rowCallback := db.Callback().Row()
	rowCallback.Register("gorm:row", RowQuery)
	rowCallback.Clauses = config.QueryClauses

	rawCallback := db.Callback().Raw()
	rawCallback.Register("gorm:raw", RawExec)
	rawCallback.Clauses = config.QueryClauses
}
