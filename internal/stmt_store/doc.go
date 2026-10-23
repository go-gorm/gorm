// Package stmt_store caches prepared SQL statements for GORM's prepared-statement
// connection wrapper. It is an internal implementation package; applications
// normally enable the wrapper through gorm.Config.PrepareStmt or a session's
// PrepareStmt option instead of constructing this store directly.
//
// # Cache and resource lifetime
//
// New creates a store keyed by exact SQL template text. Bound parameter values are
// not part of the key. Non-positive capacity defaults to math.MaxInt, and
// non-positive TTL defaults to 24 hours. Insertions and replacements set expiry;
// successful lookups update recency without renewing TTL. Keys filters expired
// entries that may still await physical cleanup.
//
// Capacity eviction, expiry cleanup, and Delete schedule Stmt.Close in a separate
// goroutine. Delete therefore does not wait for the underlying statement to close.
// Set replacing an existing key does not close its previous value. Callers must
// coordinate replacement ownership and must not directly close a shared cached
// statement while another operation intends to reuse it. The store does not own
// or close the connection pool that prepares statements.
//
// # Preparation coordination
//
// Store.New requires an already-held outer locker. It publishes an in-progress
// entry, unlocks that locker, and calls PrepareContext outside the outer lock.
// The caller must not unlock the locker again. A completion channel publishes the
// prepared statement or preparation failure to concurrent Get and Close calls.
// Get waits for preparation and callers must inspect Stmt.Error before use; that
// wait itself is not cancelled through a context. On failure, New removes the
// entry and returns an error, which must be checked before using its result.
//
// Transaction marks preparation provenance. This package records the flag; the
// outer GORM wrapper decides whether an entry is suitable for the current pool
// or transaction. A zero Stmt is not a usable cache entry because it lacks the
// completion channel. The store's TTL worker has no exposed stop operation.
package stmt_store
