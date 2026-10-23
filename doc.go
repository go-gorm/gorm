// Package gorm maps Go models to relational database operations through a
// database-specific Dialector. It provides traditional *DB chains and the typed
// G API. Both use the same schema, SQL clauses, callbacks, and connection pool.
//
// # Opening and owning a database
//
// Import a driver separately and call Open once for a reusable root handle.
// Check the initialization error. DB.DB exposes the underlying database/sql pool
// for pool limits, statistics, and shutdown; a GORM session is not a new pool.
// Configure plugins, callbacks, and naming before using the handle concurrently.
//
// # Building and executing operations
//
// Traditional chain methods such as Model, Where, Select, and Preload prepare
// statement state. Finishers such as Find, Create, Updates, and Delete execute
// callbacks and return a handle whose Error and RowsAffected must be inspected.
// For example, with an already opened db and an application model User:
//
//	var users []User
//	result := db.WithContext(ctx).Where("active = ?", true).Find(&users)
//	if result.Error != nil {
//		return result.Error
//	}
//
// G uses a model type and accepts context on each operation:
//
//	users, err := gorm.G[User](db).Where("active = ?", true).Find(ctx)
//
// First, Take, and Last report ErrRecordNotFound when no row matches. Find
// normally returns an empty result without that error. Use errors.Is to classify
// GORM sentinels; native driver errors remain possible. TranslateError requires
// a supporting dialector and does not guarantee translation of every error.
//
// # Reuse and concurrency
//
// The root handle from Open can start independent operations. An initialized
// traditional chain carries mutable statement state: do not reuse it for
// independent queries or mutate it concurrently. Derive a reusable query base
// with Session(&Session{}) to preserve conditions, or NewDB: true to discard
// them on subsequent chains. Initialized: true removes that isolation boundary.
// Sessions still share their pool or transaction and referenced configuration.
//
// # Models and writes
//
// Models are structs with exported mapped fields and optional gorm tags. Model
// is an optional convenience embedding; custom structs can define their own
// keys and timestamps. Struct conditions and struct Updates normally omit zero
// values. Use maps or explicit Select fields when zero, false, or empty strings
// must be included. Create can write generated keys and timestamps back into
// pointer destinations and can save associations.
//
// Writes normally run in a transaction per operation. Use DB.Transaction for
// multiple operations that must succeed together, use its callback's tx handle
// throughout, and return errors to trigger rollback. Check the transaction's
// final error, including commit failure. SkipDefaultTransaction removes the
// per-operation protection; it does not make multi-statement writes atomic.
//
// UPDATE and DELETE without a user condition are rejected by default. A non-zero
// model primary key can supply that condition. AllowGlobalUpdate intentionally
// disables the guard. A field of a soft-delete-capable type such as gorm.DeletedAt
// enables soft deletion unless Unscoped. A field name alone does not enable it;
// models without such a type are physically deleted.
//
// # SQL and resources
//
// Bind external data through placeholders. SQL templates, identifiers, ordering,
// and Expr text must be trusted or validated separately. Raw and Exec use SQL as
// supplied and do not automatically add model scopes or the missing-WHERE guard.
// DryRun builds SQL and Vars for inspection but still invokes callbacks. ToSQL
// and logged SQL interpolate values for diagnostics and are not safe substitutes
// for parameterized execution.
//
// Rows returns an owned result set: close it and check its iteration error.
// Row defers database errors until Scan. Normal Find and Scan manage their own
// result sets. Use contexts with deadlines where operations must be bounded,
// and call the cancel function when creating a cancellable context.
//
// # Extension points and further reading
//
// clause provides composable SQL expressions; schema parses model metadata;
// callbacks implements the default operation pipeline; migrator supports
// dialect-specific schema changes; logger controls traces. Dialectors and plugins
// adapt these components rather than changing application model code.
//
// See https://gorm.io/docs/ for usage guides, especially method_chaining.html,
// session.html, transactions.html, security.html, and write_plugins.html.
package gorm
