package callbacks

import "gorm.io/gorm"

// BeforeCreateInterface is implemented by a model that needs a BeforeCreate hook. The
// default callbacks invoke it before inserting each model, after BeforeSave. Normally
// implement the method on a pointer receiver. SkipHooks suppresses it; returning an
// error records an operation error and causes an owned default write transaction to
// roll back.
type BeforeCreateInterface interface {
	// BeforeCreate runs before inserting each model, after BeforeSave. tx shares the
	// operation's connection or transaction but starts without its query conditions; use it
	// for related work that must participate in the operation. Return an error to report
	// failure. An after hook is not an after-commit notification.
	BeforeCreate(*gorm.DB) error
}

// AfterCreateInterface is implemented by a model that needs a AfterCreate hook. The
// default callbacks invoke it after insertion and association saving, before AfterSave
// and the default transaction commit. Normally implement the method on a pointer
// receiver. SkipHooks suppresses it; returning an error records an operation error and
// causes an owned default write transaction to roll back.
type AfterCreateInterface interface {
	// AfterCreate runs after insertion and association saving, before AfterSave and the
	// default transaction commit. tx shares the operation's connection or transaction but
	// starts without its query conditions; use it for related work that must participate in
	// the operation. Return an error to report failure. An after hook is not an
	// after-commit notification.
	AfterCreate(*gorm.DB) error
}

// BeforeUpdateInterface is implemented by a model that needs a BeforeUpdate hook. The
// default callbacks invoke it before updating each model, after BeforeSave. Normally
// implement the method on a pointer receiver. SkipHooks suppresses it; returning an
// error records an operation error and causes an owned default write transaction to
// roll back.
type BeforeUpdateInterface interface {
	// BeforeUpdate runs before updating each model, after BeforeSave. tx shares the
	// operation's connection or transaction but starts without its query conditions; use it
	// for related work that must participate in the operation. Return an error to report
	// failure. An after hook is not an after-commit notification.
	BeforeUpdate(*gorm.DB) error
}

// AfterUpdateInterface is implemented by a model that needs a AfterUpdate hook. The
// default callbacks invoke it after the update and association saving, before AfterSave
// and the default transaction commit. Normally implement the method on a pointer
// receiver. SkipHooks suppresses it; returning an error records an operation error and
// causes an owned default write transaction to roll back.
type AfterUpdateInterface interface {
	// AfterUpdate runs after the update and association saving, before AfterSave and the
	// default transaction commit. tx shares the operation's connection or transaction but
	// starts without its query conditions; use it for related work that must participate in
	// the operation. Return an error to report failure. An after hook is not an
	// after-commit notification.
	AfterUpdate(*gorm.DB) error
}

// BeforeSaveInterface is implemented by a model that needs a BeforeSave hook. The
// default callbacks invoke it before either create or update, before the
// operation-specific before hook. Normally implement the method on a pointer receiver.
// SkipHooks suppresses it; returning an error records an operation error and causes an
// owned default write transaction to roll back.
type BeforeSaveInterface interface {
	// BeforeSave runs before either create or update, before the operation-specific before
	// hook. tx shares the operation's connection or transaction but starts without its
	// query conditions; use it for related work that must participate in the operation.
	// Return an error to report failure. An after hook is not an after-commit notification.
	BeforeSave(*gorm.DB) error
}

// AfterSaveInterface is implemented by a model that needs a AfterSave hook. The default
// callbacks invoke it after either create or update and its operation-specific after
// hook, before the default transaction commit. Normally implement the method on a
// pointer receiver. SkipHooks suppresses it; returning an error records an operation
// error and causes an owned default write transaction to roll back.
type AfterSaveInterface interface {
	// AfterSave runs after either create or update and its operation-specific after hook,
	// before the default transaction commit. tx shares the operation's connection or
	// transaction but starts without its query conditions; use it for related work that
	// must participate in the operation. Return an error to report failure. An after hook
	// is not an after-commit notification.
	AfterSave(*gorm.DB) error
}

// BeforeDeleteInterface is implemented by a model that needs a BeforeDelete hook. The
// default callbacks invoke it before selected association deletion and owner deletion.
// Normally implement the method on a pointer receiver. SkipHooks suppresses it;
// returning an error records an operation error and causes an owned default write
// transaction to roll back.
type BeforeDeleteInterface interface {
	// BeforeDelete runs before selected association deletion and owner deletion. tx shares
	// the operation's connection or transaction but starts without its query conditions;
	// use it for related work that must participate in the operation. Return an error to
	// report failure. An after hook is not an after-commit notification.
	BeforeDelete(*gorm.DB) error
}

// AfterDeleteInterface is implemented by a model that needs a AfterDelete hook. The
// default callbacks invoke it after owner deletion, before the default transaction
// commit. Normally implement the method on a pointer receiver. SkipHooks suppresses it;
// returning an error records an operation error and causes an owned default write
// transaction to roll back.
type AfterDeleteInterface interface {
	// AfterDelete runs after owner deletion, before the default transaction commit. tx
	// shares the operation's connection or transaction but starts without its query
	// conditions; use it for related work that must participate in the operation. Return an
	// error to report failure. An after hook is not an after-commit notification.
	AfterDelete(*gorm.DB) error
}

// AfterFindInterface is implemented by a model that needs an AfterFind hook. The
// default callbacks invoke it after a successful model query and preloading when rows
// were found. Normally implement the method on a pointer receiver. SkipHooks suppresses
// it. Returning an error records a query error; the query does not start a default
// write transaction. To roll back an enclosing Transaction, propagate that query error
// from its callback.
type AfterFindInterface interface {
	// AfterFind runs after a successful model query and preloading when rows were found. tx
	// shares the operation's connection or transaction but starts without its query
	// conditions; use it for related work that must participate in the operation. Return an
	// error to report failure. An after hook is not an after-commit notification.
	AfterFind(*gorm.DB) error
}
