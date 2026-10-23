package gorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// result collects execution metadata for WithResult. A statement modifier resets
// it before use; it is mutable operation state and must not be shared by concurrent
// operations. Not every execution path supplies a native sql.Result.
type result struct {
	// Result holds native driver metadata when execution provides it. It can be nil, for
	// example for a query or a write using RETURNING; check before calling LastInsertId or
	// RowsAffected.
	Result sql.Result
	// RowsAffected is the count recorded by the callback that last populated this
	// collector. Batched operations can overwrite it; it is not a guaranteed aggregate
	// across statements.
	RowsAffected int64
	// Error records errors propagated through DB.AddError for the attached statement.
	// Always check the operation's returned error as the authoritative completion result.
	Error error
}

// ModifyStatement clears previously collected metadata and attaches this collector to
// stmt. Reusing it overwrites earlier results.
func (info *result) ModifyStatement(stmt *Statement) {
	info.Result = nil
	info.RowsAffected = 0
	info.Error = nil
	stmt.Result = info
}

// Build emits no SQL. result satisfies clause.Expression so it can be passed to G or
// DB.Clauses; ModifyStatement performs the actual attachment.
func (result) Build(clause.Builder) {
}

// WithResult allocates an execution-metadata collector to pass as an option to G
// or DB.Clauses. Its exported Result, RowsAffected, and Error fields can be inspected
// after an operation. Check the operation error first and test Result for nil before
// using driver-specific metadata. Allocate a separate collector for concurrent work;
// reuse resets its contents, and batches need not report an aggregate in it.
//
//	info := gorm.WithResult()
//	err := gorm.G[User](db, info).Create(ctx, &user)
//	if err != nil {
//		return err
//	}
//	fmt.Println(info.RowsAffected)
func WithResult() *result {
	return &result{}
}

// Interface is the entry point returned by G for model type T. It combines typed
// creation and query chains with raw SQL. Execution methods accept context and
// return errors directly; Save and FirstOrCreate are intentionally not included.
// A new G call is needed to start a raw statement after entering a query chain.
type Interface[T any] interface {
	// Raw records a trusted SQL template and bound values, returning a query execution
	// interface. Finish with Find, Scan, Row, or Rows. SQL is used as supplied, without
	// automatically adding model predicates or soft-delete filters.
	//
	// Create a fresh G[T](db) entry point for each independent Raw builder. Raw currently
	// appends into the entry point's shared operation backing array, so deriving another
	// Raw builder from the same entry point can replace the earlier builder's SQL.
	Raw(sql string, values ...interface{}) ExecInterface[T]
	// Exec executes a trusted SQL template with bound values and returns the error
	// directly. It does not add model scopes, write hooks, or a missing-WHERE guard. Pass a
	// separate WithResult collector to G when affected rows or driver metadata are needed.
	Exec(ctx context.Context, sql string, values ...interface{}) error
	CreateInterface[T]
}

// CreateInterface exposes operations available before entering a query-only chain.
// Table, Select, and Omit preserve the ability to Create; methods such as Where
// return ChainInterface, whose Set result only permits Update. The model type T
// sets schema and result typing; SQL expressions still require trusted templates.
type CreateInterface[T any] interface {
	ExecInterface[T]
	// Scopes records callbacks that directly modify the execution Statement in order. These
	// use func(*Statement), not the traditional func(*DB) *DB scope signature. Keep
	// callbacks safe for each execution; they run when operations are applied.
	Scopes(scopes ...func(db *Statement)) ChainInterface[T]
	// Where records AND predicates using DB.Where rules. Struct conditions omit zero
	// values; use maps or explicit expressions for zero-valued predicates. Bind external
	// values through args.
	Where(query interface{}, args ...interface{}) ChainInterface[T]
	// Not records negated predicates using DB.Not rules and the same input formats as
	// Where.
	Not(query interface{}, args ...interface{}) ChainInterface[T]
	// Or records a disjunction using DB.Or. Use nested clause expressions to specify
	// grouping when mixing AND and OR.
	Or(query interface{}, args ...interface{}) ChainInterface[T]
	// Limit records a row limit; -1 cancels an earlier limit. A fresh zero limit
	// requests no rows, but zero preserves an existing non-nil limit during merging.
	// Use a fresh builder for that case and deterministic ordering for pagination.
	Limit(offset int) ChainInterface[T]
	// Offset records the number of rows to skip; -1 cancels an earlier offset. Stable
	// pagination requires deterministic ordering.
	Offset(offset int) ChainInterface[T]
	// Joins records a typed association join and an optional callback for ON predicates and
	// field selection. The callback receives the join and current table references and can
	// return an execution error. Use clause.LeftJoin.Association or clause.Has to choose
	// join semantics. It runs when the chain is applied, not when Joins is called.
	Joins(query clause.JoinTarget, on func(db JoinBuilder, joinTable clause.Table, curTable clause.Table) error) ChainInterface[T]
	// Preload records a model association path and an optional callback customizing its
	// separate query. Callback errors are recorded on the operation. Nested paths are
	// supported; filters affect associated rows rather than parent rows. Keep relationship
	// keys in the selected columns.
	Preload(association string, query func(db PreloadBuilder) error) ChainInterface[T]
	// Select records field selection or a trusted SQL expression with bound args.
	// Explicitly selected fields can include zero values in Updates; Omit and field
	// permissions still apply.
	Select(query string, args ...interface{}) CreateInterface[T]
	// Omit records fields or associations to exclude from subsequent queries or writes. It
	// replaces the omit list; see DB.Omit for association wildcard behavior.
	Omit(columns ...string) CreateInterface[T]
	// MapColumns records a mapping from returned column names to model fields for scanning.
	// It retains the map; do not mutate it during execution.
	MapColumns(m map[string]string) ChainInterface[T]
	// Distinct records DISTINCT and, when supplied, selected fields. Use a trusted field
	// selection; SQL expressions are not validated.
	Distinct(args ...interface{}) ChainInterface[T]
	// Group records a GROUP BY column or trusted expression. Pair it with an aggregate
	// Select and optional Having predicate.
	Group(name string) ChainInterface[T]
	// Having records post-group predicates using DB.Having rules. Bind values separately
	// from the trusted expression text.
	Having(query interface{}, args ...interface{}) ChainInterface[T]
	// Order records sorting using a trusted SQL string or clause.OrderBy/OrderByColumn.
	// Validate dynamic identifiers and directions before constructing them.
	Order(value interface{}) ChainInterface[T]
	// Build renders this query as a subquery into a *Statement builder, rebinding
	// placeholders into its variable sequence. It uses DryRun query callbacks and emits
	// nothing for other builder implementations; it does not fetch rows.
	Build(builder clause.Builder)

	// Delete removes rows for model T matching the chain and returns the affected-row count
	// and error. It starts from a zero-valued model, so normally provide Where conditions.
	// DeletedAt uses soft deletion; the missing-WHERE guard still applies.
	Delete(ctx context.Context) (rowsAffected int, err error)
	// Update writes one column for model T matching the chain and returns affected rows and
	// an error. Model hooks and update timestamps apply. Supply conditions unless a global
	// update is intended and explicitly enabled.
	//
	// For struct-valued T, this path uses a non-addressable model value. A pointer-receiver
	// BeforeSave, BeforeUpdate, AfterUpdate, or AfterSave hook can cause ErrInvalidValue
	// rather than being invoked. Use the traditional Model(&model).Update/Updates path when
	// addressable update hooks are required.
	Update(ctx context.Context, name string, value any) (rowsAffected int, err error)
	// Updates writes fields from t using DB.Updates rules and returns affected rows and an
	// error. Struct zero values are omitted unless selected; map-valued T includes zero
	// values. Conditions or a non-zero model key must restrict the write.
	//
	// For struct-valued T, this path uses a non-addressable model value. A pointer-receiver
	// BeforeSave, BeforeUpdate, AfterUpdate, or AfterSave hook can cause ErrInvalidValue
	// rather than being invoked. Use the traditional Model(&model).Update/Updates path when
	// addressable update hooks are required.
	Updates(ctx context.Context, t T) (rowsAffected int, err error)
	// Count returns an aggregate for model T and the chain. Use "*" for all rows or a
	// single column name to count non-NULL values. Distinct counts unique column values;
	// Group returns the number of groups. The string is passed through Select and DB.Count:
	// it is not an arbitrary expression argument. An explicit trusted count(...) selection
	// is retained, while unsupported selections can fall back to count(*).
	Count(ctx context.Context, column string) (result int64, err error)

	// Table records a table override using DB.Table rules. Simple names are quoted; SQL
	// expressions and identifiers must be trusted. It does not execute SQL.
	Table(name string, args ...interface{}) CreateInterface[T]
	// Create inserts r and returns the operation error directly. Pass a non-nil *T;
	// generated keys and timestamps are written back. It runs normal create callbacks,
	// including hooks and included associations. Use WithResult when execution metadata is
	// needed.
	Create(ctx context.Context, r *T) error
	// CreateInBatches inserts the slice referenced by r in batches of a positive batchSize
	// and writes generated values back. It returns the first error. The traditional
	// DB.CreateInBatches transaction rules apply; disabling default transactions allows
	// partial persistence.
	CreateInBatches(ctx context.Context, r *[]T, batchSize int) error
	// Set collects column and association assignments and returns a builder that can Create
	// or Update. It does not execute SQL. Association operations run before owner
	// assignments; use an explicit transaction for atomicity across the whole operation.
	//
	// Current many-to-many association unlink, delete, and update paths rebuild the owner
	// EXISTS query without the chain's owner filters. They can therefore affect links or
	// related rows belonging to other owners; Association.Conditions filters related
	// records, not owners. Use explicit owner-key-qualified operations when owner scoping
	// is required. An enclosing transaction provides atomicity but does not restore those
	// filters.
	Set(assignments ...clause.Assigner) SetCreateOrUpdateInterface[T]
}

// ChainInterface describes a typed query or update/delete chain. Chain methods
// record operations for later execution and preserve the existing builder's operation
// list. Finishers apply the operations and return database errors directly.
// Referenced maps, expression values, and callbacks are not deep-copied; do not
// mutate them while executing a builder.
type ChainInterface[T any] interface {
	ExecInterface[T]
	// Scopes records callbacks that directly modify the execution Statement in order. These
	// use func(*Statement), not the traditional func(*DB) *DB scope signature. Keep
	// callbacks safe for each execution; they run when operations are applied.
	Scopes(scopes ...func(db *Statement)) ChainInterface[T]
	// Where records AND predicates using DB.Where rules. Struct conditions omit zero
	// values; use maps or explicit expressions for zero-valued predicates. Bind external
	// values through args.
	Where(query interface{}, args ...interface{}) ChainInterface[T]
	// Not records negated predicates using DB.Not rules and the same input formats as
	// Where.
	Not(query interface{}, args ...interface{}) ChainInterface[T]
	// Or records a disjunction using DB.Or. Use nested clause expressions to specify
	// grouping when mixing AND and OR.
	Or(query interface{}, args ...interface{}) ChainInterface[T]
	// Limit records a row limit; -1 cancels an earlier limit. A fresh zero limit
	// requests no rows, but zero preserves an existing non-nil limit during merging.
	// Use a fresh builder for that case and deterministic ordering for pagination.
	Limit(offset int) ChainInterface[T]
	// Offset records the number of rows to skip; -1 cancels an earlier offset. Stable
	// pagination requires deterministic ordering.
	Offset(offset int) ChainInterface[T]
	// Joins records a typed association join and an optional callback for ON predicates and
	// field selection. The callback receives the join and current table references and can
	// return an execution error. Use clause.LeftJoin.Association or clause.Has to choose
	// join semantics. It runs when the chain is applied, not when Joins is called.
	Joins(query clause.JoinTarget, on func(db JoinBuilder, joinTable clause.Table, curTable clause.Table) error) ChainInterface[T]
	// Preload records a model association path and an optional callback customizing its
	// separate query. Callback errors are recorded on the operation. Nested paths are
	// supported; filters affect associated rows rather than parent rows. Keep relationship
	// keys in the selected columns.
	Preload(association string, query func(db PreloadBuilder) error) ChainInterface[T]
	// Select records field selection or a trusted SQL expression with bound args.
	// Explicitly selected fields can include zero values in Updates; Omit and field
	// permissions still apply.
	Select(query string, args ...interface{}) ChainInterface[T]
	// Omit records fields or associations to exclude from subsequent queries or writes. It
	// replaces the omit list; see DB.Omit for association wildcard behavior.
	Omit(columns ...string) ChainInterface[T]
	// MapColumns records a mapping from returned column names to model fields for scanning.
	// It retains the map; do not mutate it during execution.
	MapColumns(m map[string]string) ChainInterface[T]
	// Distinct records DISTINCT and, when supplied, selected fields. Use a trusted field
	// selection; SQL expressions are not validated.
	Distinct(args ...interface{}) ChainInterface[T]
	// Group records a GROUP BY column or trusted expression. Pair it with an aggregate
	// Select and optional Having predicate.
	Group(name string) ChainInterface[T]
	// Having records post-group predicates using DB.Having rules. Bind values separately
	// from the trusted expression text.
	Having(query interface{}, args ...interface{}) ChainInterface[T]
	// Order records sorting using a trusted SQL string or clause.OrderBy/OrderByColumn.
	// Validate dynamic identifiers and directions before constructing them.
	Order(value interface{}) ChainInterface[T]
	// Set collects column and association assignments for Update on the current query
	// chain. It does not execute SQL and does not expose Create after chaining. Association
	// operations run before owner assignments; use an explicit transaction for atomicity
	// across them.
	//
	// Current many-to-many association unlink, delete, and update paths rebuild the owner
	// EXISTS query without the chain's owner filters. They can therefore affect links or
	// related rows belonging to other owners; Association.Conditions filters related
	// records, not owners. Use explicit owner-key-qualified operations when owner scoping
	// is required. An enclosing transaction provides atomicity but does not restore those
	// filters.
	Set(assignments ...clause.Assigner) SetUpdateOnlyInterface[T]

	// Build renders this query as a subquery into a *Statement builder, rebinding
	// placeholders into its variable sequence. It uses DryRun query callbacks and emits
	// nothing for other builder implementations; it does not fetch rows.
	Build(builder clause.Builder)

	// Table records a table override using DB.Table rules. Simple names are quoted; SQL
	// expressions and identifiers must be trusted. It does not execute SQL.
	Table(name string, args ...interface{}) ChainInterface[T]
	// Delete removes rows for model T matching the chain and returns the affected-row count
	// and error. It starts from a zero-valued model, so normally provide Where conditions.
	// A soft-delete-capable field type such as gorm.DeletedAt enables soft deletion; the
	// field name alone does not. The missing-WHERE guard still applies.
	Delete(ctx context.Context) (rowsAffected int, err error)
	// Update writes one column for model T matching the chain and returns affected rows and
	// an error. Model hooks and update timestamps apply. Supply conditions unless a global
	// update is intended and explicitly enabled.
	//
	// For struct-valued T, this path uses a non-addressable model value. A pointer-receiver
	// BeforeSave, BeforeUpdate, AfterUpdate, or AfterSave hook can cause ErrInvalidValue
	// rather than being invoked. Use the traditional Model(&model).Update/Updates path when
	// addressable update hooks are required.
	Update(ctx context.Context, name string, value any) (rowsAffected int, err error)
	// Updates writes fields from t using DB.Updates rules and returns affected rows and an
	// error. Struct zero values are omitted unless selected; map-valued T includes zero
	// values. Conditions or a non-zero model key must restrict the write.
	//
	// For struct-valued T, this path uses a non-addressable model value. A pointer-receiver
	// BeforeSave, BeforeUpdate, AfterUpdate, or AfterSave hook can cause ErrInvalidValue
	// rather than being invoked. Use the traditional Model(&model).Update/Updates path when
	// addressable update hooks are required.
	Updates(ctx context.Context, t T) (rowsAffected int, err error)
	// Count returns an aggregate for model T and the chain. Use "*" for all rows or a
	// single column name to count non-NULL values. Distinct counts unique column values;
	// Group returns the number of groups. The string is passed through Select and DB.Count:
	// it is not an arbitrary expression argument. An explicit trusted count(...) selection
	// is retained, while unsupported selections can fall back to count(*).
	Count(ctx context.Context, column string) (result int64, err error)
}

// SetUpdateOnlyInterface executes assignments attached to a query chain. Only Update is
// exposed; use an initial CreateInterface.Set when constructing an insert from
// assignments. Owner-column assignments use the chain's predicates.
//
// Current many-to-many association unlink, delete, and update paths rebuild the owner
// EXISTS query without the chain's owner filters. They can therefore affect links or
// related rows belonging to other owners; Association.Conditions filters related
// records, not owners. Use explicit owner-key-qualified operations when owner scoping
// is required. An enclosing transaction provides atomicity but does not restore those
// filters.
type SetUpdateOnlyInterface[T any] interface {
	// Update executes association operations first and then owner-column assignments,
	// stopping on the first error. The returned row count covers the owner update, not all
	// association changes; association-only work returns zero. Separate operations are not
	// automatically wrapped in one transaction.
	//
	// Current many-to-many association unlink, delete, and update paths rebuild the owner
	// EXISTS query without the chain's owner filters. They can therefore affect links or
	// related rows belonging to other owners; Association.Conditions filters related
	// records, not owners. Use explicit owner-key-qualified operations when owner scoping
	// is required. An enclosing transaction provides atomicity but does not restore those
	// filters.
	//
	// For struct-valued T, the owner update uses a non-addressable model value;
	// pointer-receiver update/save hooks can produce ErrInvalidValue. Use traditional
	// Model(&model) updates when those hooks are required.
	Update(ctx context.Context) (rowsAffected int, err error)
}

// SetCreateOrUpdateInterface can execute a root Set as either Create or Update.
// Assignments may include related-record operations. Multiple association and owner
// operations are not automatically one atomic unit; use DB.Transaction when needed.
//
// Current many-to-many association unlink, delete, and update paths rebuild the owner
// EXISTS query without the chain's owner filters. They can therefore affect links or
// related rows belonging to other owners; Association.Conditions filters related
// records, not owners. Use explicit owner-key-qualified operations when owner scoping
// is required. An enclosing transaction provides atomicity but does not restore those
// filters.
type SetCreateOrUpdateInterface[T any] interface {
	// Create executes association operations and then inserts a map built from owner-column
	// assignments. Map-create rules apply: no model hook calls or model key backfill.
	// Association-only work performs no owner insert. Use a transaction when all operations
	// must succeed together.
	Create(ctx context.Context) error
	// Update executes association operations first and then owner-column assignments,
	// stopping on the first error. The returned row count covers the owner update, not all
	// association changes; association-only work returns zero. Separate operations are not
	// automatically wrapped in one transaction.
	//
	// Current many-to-many association unlink, delete, and update paths rebuild the owner
	// EXISTS query without the chain's owner filters. They can therefore affect links or
	// related rows belonging to other owners; Association.Conditions filters related
	// records, not owners. Use explicit owner-key-qualified operations when owner scoping
	// is required. An enclosing transaction provides atomicity but does not restore those
	// filters.
	//
	// For struct-valued T, the owner update uses a non-addressable model value;
	// pointer-receiver update/save hooks can produce ErrInvalidValue. Use traditional
	// Model(&model) updates when those hooks are required.
	Update(ctx context.Context) (rowsAffected int, err error)
}

// ExecInterface executes a typed query using a context supplied by the caller.
// First, Last, and Take return a single T and ErrRecordNotFound on a miss; Find
// returns a slice without a not-found error. Rows transfers result-set ownership
// to the caller. Check every returned error before using data.
type ExecInterface[T any] interface {
	// Scan executes the typed model query into a writable destination such as a projection
	// struct or slice. This generic method uses the normal Find callback pipeline, unlike
	// traditional DB.Scan, which uses row callbacks. It returns the execution error
	// directly.
	Scan(ctx context.Context, r interface{}) error
	// First returns one T using ascending primary-key ordering, or ErrRecordNotFound when
	// no row matches. Check err before using the returned value. Existing chain ordering
	// and conditions still participate.
	First(context.Context) (T, error)
	// Last returns one T using descending primary-key ordering, or ErrRecordNotFound on a
	// miss. Primary-key order is not necessarily creation-time order.
	Last(ctx context.Context) (T, error)
	// Take returns one T without adding an order; an existing Order is retained. It returns
	// ErrRecordNotFound when nothing matches.
	Take(context.Context) (T, error)
	// Find returns matching rows as []T and an error. An empty result normally has no
	// error; len(result) reports whether anything matched.
	Find(ctx context.Context) ([]T, error)
	// FindInBatches calls fc for successive non-empty model batches; numbering starts at 1
	// and batchSize must be positive. Include the prioritized primary key for continuation
	// and do not change it during fc. The batch slice may be reused: copy data that must
	// outlive the callback. Query or callback errors stop iteration; no snapshot
	// transaction is created automatically.
	//
	// Traversal uses an ascending prioritized-primary-key cursor. Avoid conflicting Order
	// clauses, including descending key order, which can skip or repeat records; see
	// DB.FindInBatches.
	FindInBatches(ctx context.Context, batchSize int, fc func(data []T, batch int) error) error
	// Row returns a *sql.Row for manual Scan; query errors and sql.ErrNoRows are reported
	// by Scan. DryRun or construction failure can leave no row. This path does not perform
	// ordinary model preloading or query hooks.
	Row(ctx context.Context) *sql.Row
	// Rows returns an owned result set or an error. On success, close the rows, iterate
	// with Next/Scan, and check Err after iteration. DryRun is unsupported; this path does
	// not perform normal query preloading or hooks.
	Rows(ctx context.Context) (*sql.Rows, error)
}

// JoinBuilder customizes a generic association join's selected fields and ON
// predicates.
// Use it only inside the Joins callback; it mutates the callback's temporary statement.
// SQL templates must be trusted. In the current implementation Or and Not both
// forward to Where; use explicit clause.Or or clause.Not expressions with Where
// when those boolean operations are required.
type JoinBuilder interface {
	// Select sets the fields to fetch for the joined or preloaded model. Keep relationship
	// key columns required to attach results to their owners.
	Select(...string) JoinBuilder
	// Omit excludes fields from the joined or preloaded model selection. Do not omit keys
	// required for relation assignment.
	Omit(...string) JoinBuilder
	// Where adds predicates to this callback-local statement. Use bound args for values;
	// struct conditions omit zero values. Pass clause expressions for explicit boolean
	// grouping.
	Where(query interface{}, args ...interface{}) JoinBuilder
	// Not currently delegates to Where in this builder and does not negate the condition.
	// To express negation here, pass a clause.Not expression to Where.
	Not(query interface{}, args ...interface{}) JoinBuilder
	// Or currently delegates to Where in this builder and therefore adds an AND condition.
	// To express a disjunction here, pass a clause.Or expression to Where.
	Or(query interface{}, args ...interface{}) JoinBuilder
}

// PreloadBuilder customizes the separate query used for a generic Preload.
// Keep association key columns in Select so fetched rows can be assigned to owners.
// Limit applies to the entire preload query; LimitPerRecord applies per owner where
// supported. In the current implementation Or and Not forward to Where; express
// those operators explicitly with clause.Or or clause.Not passed to Where.
type PreloadBuilder interface {
	// Select sets the fields to fetch for the joined or preloaded model. Keep relationship
	// key columns required to attach results to their owners.
	Select(...string) PreloadBuilder
	// Omit excludes fields from the joined or preloaded model selection. Do not omit keys
	// required for relation assignment.
	Omit(...string) PreloadBuilder
	// Where adds predicates to this callback-local statement. Use bound args for values;
	// struct conditions omit zero values. Pass clause expressions for explicit boolean
	// grouping.
	Where(query interface{}, args ...interface{}) PreloadBuilder
	// Not currently delegates to Where in this builder and does not negate the condition.
	// To express negation here, pass a clause.Not expression to Where.
	Not(query interface{}, args ...interface{}) PreloadBuilder
	// Or currently delegates to Where in this builder and therefore adds an AND condition.
	// To express a disjunction here, pass a clause.Or expression to Where.
	Or(query interface{}, args ...interface{}) PreloadBuilder
	// Limit bounds the entire association query, not the number of related rows per owner.
	// Use LimitPerRecord for a supported per-owner limit.
	Limit(offset int) PreloadBuilder
	// Offset skips rows in the overall association query; it is not a per-owner offset. Use
	// deterministic ordering when applying it.
	Offset(offset int) PreloadBuilder
	// Order sorts association rows using trusted SQL or order clauses. It also supplies
	// ordering for LimitPerRecord when present.
	Order(value interface{}) PreloadBuilder
	// LimitPerRecord requests at most num rows per owner when num is positive. It uses
	// ROW_NUMBER partitioned by owner foreign keys and requires database window-function
	// support. Many-to-many relations return an error; relations without owner-key
	// partitions do not apply the limit. Without Order the window defaults to descending
	// primary key.
	LimitPerRecord(num int) PreloadBuilder
}

// op is a deferred transformation of a traditional DB handle. Generic chains replay
// their operations on execution rather than issuing SQL while being built.
type op func(*DB) *DB

// G creates a typed entry point for model T using an existing DB and optional clause
// expressions or statement modifiers, such as clause.OnConflict or WithResult().
// Use a non-pointer model type (for example G[User]) and pass context to finishers:
//
//	users, err := gorm.G[User](db).Where("active = ?", true).Find(ctx)
//
// Normal execution starts a NewDB session on the supplied pool or transaction and
// replays the generic chain. Put query conditions on that chain instead of assuming
// conditions on an initialized traditional DB will be retained. To work inside a
// transaction, call G with its tx handle. DryRun retains the underlying statement
// and context instead of taking this normal fresh-session path; do not share a
// DryRun builder concurrently or assume its execution context replaces the session's.
//
// See https://gorm.io/docs/the_generics_way.html.
//
// Use a fresh G call for each independent Raw builder: Raw currently shares
// operation-slice storage with its entry point, and a later Raw call on that entry
// point can overwrite an earlier builder's SQL.
//
// For struct-valued T, generic Update and Updates use a non-addressable model value and
// can fail with ErrInvalidValue when pointer-receiver update/save hooks are present.
// Use traditional Model(&model) updates for those hooks.
func G[T any](db *DB, opts ...clause.Expression) Interface[T] {
	v := &g[T]{
		db:  db,
		ops: make([]op, 0, 5),
	}

	if len(opts) > 0 {
		v.ops = append(v.ops, func(db *DB) *DB {
			return db.Clauses(opts...)
		})
	}

	v.createG = &createG[T]{
		chainG: chainG[T]{
			execG: execG[T]{g: v},
		},
	}
	return v
}

// g stores the underlying DB and deferred operations for a typed entry point. Execution
// state belongs to the applied traditional session, not the type parameter.
type g[T any] struct {
	*createG[T]
	// db is the source of configuration and the pool or transaction used during apply.
	db *DB
	// ops is the ordered list of deferred statement transformations; captured inputs are
	// not deep-copied.
	ops []op
}

// apply creates a NewDB session with ctx for normal execution and replays recorded
// operations. DryRun deliberately uses the stored DB directly, preserving its statement
// and context.
func (g *g[T]) apply(ctx context.Context) *DB {
	db := g.db
	if !db.DryRun {
		db = db.Session(&Session{NewDB: true, Context: ctx}).getInstance()
	}

	for _, op := range g.ops {
		db = op(db)
	}
	return db
}

// Raw records a trusted SQL template and bound values, returning a query execution
// interface. Finish with Find, Scan, Row, or Rows. SQL is used as supplied, without
// automatically adding model predicates or soft-delete filters.
//
// Create a fresh G[T](db) entry point for each independent Raw builder. Raw currently
// appends into the entry point's shared operation backing array, so deriving another
// Raw builder from the same entry point can replace the earlier builder's SQL.
func (c *g[T]) Raw(sql string, values ...interface{}) ExecInterface[T] {
	return execG[T]{g: &g[T]{
		db: c.db,
		ops: append(c.ops, func(db *DB) *DB {
			var r T
			return db.Model(r).Raw(sql, values...)
		}),
	}}
}

// Exec executes a trusted SQL template with bound values and returns the error
// directly. It does not add model scopes, write hooks, or a missing-WHERE guard. Pass a
// separate WithResult collector to G when affected rows or driver metadata are needed.
func (c *g[T]) Exec(ctx context.Context, sql string, values ...interface{}) error {
	var r T
	return c.apply(ctx).Model(r).Exec(sql, values...).Error
}

// createG preserves creation methods while embedding the generic chain implementation.
// Table, Select, and Omit return this capability-preserving wrapper.
type createG[T any] struct {
	chainG[T]
}

// Table records a table override using DB.Table rules. Simple names are quoted; SQL
// expressions and identifiers must be trusted. It does not execute SQL.
// The returned interface retains Create and CreateInBatches.
func (c createG[T]) Table(name string, args ...interface{}) CreateInterface[T] {
	return createG[T]{c.with(func(db *DB) *DB {
		return db.Table(name, args...)
	})}
}

// Select records field selection or a trusted SQL expression with bound args.
// Explicitly selected fields can include zero values in Updates; Omit and field
// permissions still apply.
// The returned interface retains Create and CreateInBatches.
func (c createG[T]) Select(query string, args ...interface{}) CreateInterface[T] {
	return createG[T]{c.with(func(db *DB) *DB {
		return db.Select(query, args...)
	})}
}

// Omit records fields or associations to exclude from subsequent queries or writes. It
// replaces the omit list; see DB.Omit for association wildcard behavior.
// The returned interface retains Create and CreateInBatches.
func (c createG[T]) Omit(columns ...string) CreateInterface[T] {
	return createG[T]{c.with(func(db *DB) *DB {
		return db.Omit(columns...)
	})}
}

// Set collects column and association assignments and returns a builder that can Create
// or Update. It does not execute SQL. Association operations run before owner
// assignments; use an explicit transaction for atomicity across the whole operation.
//
// Current many-to-many association unlink, delete, and update paths rebuild the owner
// EXISTS query without the chain's owner filters. They can therefore affect links or
// related rows belonging to other owners; Association.Conditions filters related
// records, not owners. Use explicit owner-key-qualified operations when owner scoping
// is required. An enclosing transaction provides atomicity but does not restore those
// filters.
func (c createG[T]) Set(assignments ...clause.Assigner) SetCreateOrUpdateInterface[T] {
	return c.processSet(assignments...)
}

// Create inserts r and returns the operation error directly. Pass a non-nil *T;
// generated keys and timestamps are written back. It runs normal create callbacks,
// including hooks and included associations. Use WithResult when execution metadata is
// needed.
func (c createG[T]) Create(ctx context.Context, r *T) error {
	return c.g.apply(ctx).Create(r).Error
}

// CreateInBatches inserts the slice referenced by r in batches of a positive batchSize
// and writes generated values back. It returns the first error. The traditional
// DB.CreateInBatches transaction rules apply; disabling default transactions allows
// partial persistence.
func (c createG[T]) CreateInBatches(ctx context.Context, r *[]T, batchSize int) error {
	return c.g.apply(ctx).CreateInBatches(r, batchSize).Error
}

// chainG records query transformations and exposes typed query/update/delete execution.
// with copies the operation slice so ordinary derived chains do not append into their
// parent.
type chainG[T any] struct {
	execG[T]
}

// getInstance applies the chain using a background context and sets model T for
// clause/subquery construction. It returns an initialized traditional handle for the
// builder.
func (c chainG[T]) getInstance() *DB {
	var r T
	return c.g.apply(context.Background()).Model(r).getInstance()
}

// with copies the operation slice and appends v for a derived chain. Captured maps,
// clauses, and callback values remain shared references.
func (c chainG[T]) with(v op) chainG[T] {
	return chainG[T]{
		execG: execG[T]{g: &g[T]{
			db:  c.g.db,
			ops: append(append([]op(nil), c.g.ops...), v),
		}},
	}
}

// Table records a table override using DB.Table rules. Simple names are quoted; SQL
// expressions and identifiers must be trusted. It does not execute SQL.
func (c chainG[T]) Table(name string, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Table(name, args...)
	})
}

// Scopes records callbacks that directly modify the execution Statement in order. These
// use func(*Statement), not the traditional func(*DB) *DB scope signature. Keep
// callbacks safe for each execution; they run when operations are applied.
func (c chainG[T]) Scopes(scopes ...func(db *Statement)) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		for _, fc := range scopes {
			fc(db.Statement)
		}
		return db
	})
}

// Where records AND predicates using DB.Where rules. Struct conditions omit zero
// values; use maps or explicit expressions for zero-valued predicates. Bind external
// values through args.
func (c chainG[T]) Where(query interface{}, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Where(query, args...)
	})
}

// Not records negated predicates using DB.Not rules and the same input formats as
// Where.
func (c chainG[T]) Not(query interface{}, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Not(query, args...)
	})
}

// Or records a disjunction using DB.Or. Use nested clause expressions to specify
// grouping when mixing AND and OR.
func (c chainG[T]) Or(query interface{}, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Or(query, args...)
	})
}

// Limit records a row limit; -1 cancels an earlier limit. A fresh zero limit
// requests no rows, but zero preserves an existing non-nil limit during merging.
// Use a fresh builder for that case and deterministic ordering for pagination.
func (c chainG[T]) Limit(offset int) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Limit(offset)
	})
}

// Offset records the number of rows to skip; -1 cancels an earlier offset. Stable
// pagination requires deterministic ordering.
func (c chainG[T]) Offset(offset int) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Offset(offset)
	})
}

// joinBuilder adapts a temporary initialized DB into the restricted join callback API.
// Its methods mutate that same callback-local statement.
type joinBuilder struct {
	// db holds the callback-local ON predicates and field selections consumed by Joins.
	db *DB
}

// Where adds predicates to this callback-local statement. Use bound args for values;
// struct conditions omit zero values. Pass clause expressions for explicit boolean
// grouping.
func (q *joinBuilder) Where(query interface{}, args ...interface{}) JoinBuilder {
	q.db.Where(query, args...)
	return q
}

// Or currently delegates to Where in this builder and therefore adds an AND condition.
// To express a disjunction here, pass a clause.Or expression to Where.
func (q *joinBuilder) Or(query interface{}, args ...interface{}) JoinBuilder {
	q.db.Where(query, args...)
	return q
}

// Not currently delegates to Where in this builder and does not negate the condition.
// To express negation here, pass a clause.Not expression to Where.
func (q *joinBuilder) Not(query interface{}, args ...interface{}) JoinBuilder {
	q.db.Where(query, args...)
	return q
}

// Select sets the fields to fetch for the joined or preloaded model. Keep relationship
// key columns required to attach results to their owners.
func (q *joinBuilder) Select(columns ...string) JoinBuilder {
	q.db.Select(columns)
	return q
}

// Omit excludes fields from the joined or preloaded model selection. Do not omit keys
// required for relation assignment.
func (q *joinBuilder) Omit(columns ...string) JoinBuilder {
	q.db.Omit(columns...)
	return q
}

// preloadBuilder collects query customization and an optional per-owner limit for a
// generic association preload.
type preloadBuilder struct {
	// limitPerRecord enables a window-function query when positive and the relation
	// supplies owner-key partitions. Many-to-many relations reject a positive limit.
	limitPerRecord int
	// db is the association query being customized; its conditions do not filter parent
	// rows.
	db *DB
}

// Where adds predicates to this callback-local statement. Use bound args for values;
// struct conditions omit zero values. Pass clause expressions for explicit boolean
// grouping.
func (q *preloadBuilder) Where(query interface{}, args ...interface{}) PreloadBuilder {
	q.db.Where(query, args...)
	return q
}

// Or currently delegates to Where in this builder and therefore adds an AND condition.
// To express a disjunction here, pass a clause.Or expression to Where.
func (q *preloadBuilder) Or(query interface{}, args ...interface{}) PreloadBuilder {
	q.db.Where(query, args...)
	return q
}

// Not currently delegates to Where in this builder and does not negate the condition.
// To express negation here, pass a clause.Not expression to Where.
func (q *preloadBuilder) Not(query interface{}, args ...interface{}) PreloadBuilder {
	q.db.Where(query, args...)
	return q
}

// Select sets the fields to fetch for the joined or preloaded model. Keep relationship
// key columns required to attach results to their owners.
func (q *preloadBuilder) Select(columns ...string) PreloadBuilder {
	q.db.Select(columns)
	return q
}

// Omit excludes fields from the joined or preloaded model selection. Do not omit keys
// required for relation assignment.
func (q *preloadBuilder) Omit(columns ...string) PreloadBuilder {
	q.db.Omit(columns...)
	return q
}

// Limit bounds the entire association query, not the number of related rows per owner.
// Use LimitPerRecord for a supported per-owner limit.
func (q *preloadBuilder) Limit(limit int) PreloadBuilder {
	q.db.Limit(limit)
	return q
}

// Offset skips rows in the overall association query; it is not a per-owner offset. Use
// deterministic ordering when applying it.
func (q *preloadBuilder) Offset(offset int) PreloadBuilder {
	q.db.Offset(offset)
	return q
}

// Order sorts association rows using trusted SQL or order clauses. It also supplies
// ordering for LimitPerRecord when present.
func (q *preloadBuilder) Order(value interface{}) PreloadBuilder {
	q.db.Order(value)
	return q
}

// LimitPerRecord requests at most num rows per owner when num is positive. It uses
// ROW_NUMBER partitioned by owner foreign keys and requires database window-function
// support. Many-to-many relations return an error; relations without owner-key
// partitions do not apply the limit. Without Order the window defaults to descending
// primary key.
func (q *preloadBuilder) LimitPerRecord(num int) PreloadBuilder {
	q.limitPerRecord = num
	return q
}

// Joins records a typed association join and an optional callback for ON predicates and
// field selection. The callback receives the join and current table references and can
// return an execution error. Use clause.LeftJoin.Association or clause.Has to choose
// join semantics. It runs when the chain is applied, not when Joins is called.
func (c chainG[T]) Joins(jt clause.JoinTarget, on func(db JoinBuilder, joinTable clause.Table, curTable clause.Table) error) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		if jt.Table == "" {
			jt.Table = clause.JoinTable(strings.Split(jt.Association, ".")...).Name
		}

		q := joinBuilder{db: db.Session(&Session{NewDB: true, Initialized: true}).Table(jt.Table)}
		if on != nil {
			if err := on(&q, clause.Table{Name: jt.Table}, clause.Table{Name: clause.CurrentTable}); err != nil {
				db.AddError(err)
			}
		}

		j := join{
			Name:     jt.Association,
			Alias:    jt.Table,
			Selects:  q.db.Statement.Selects,
			Omits:    q.db.Statement.Omits,
			JoinType: jt.Type,
		}

		if where, ok := q.db.Statement.Clauses["WHERE"].Expression.(clause.Where); ok {
			j.On = &where
		}

		if jt.Subquery != nil {
			joinType := j.JoinType
			if joinType == "" {
				joinType = clause.LeftJoin
			}

			if db, ok := jt.Subquery.(interface{ getInstance() *DB }); ok {
				stmt := db.getInstance().Statement
				if len(j.Selects) == 0 {
					j.Selects = stmt.Selects
				}
				if len(j.Omits) == 0 {
					j.Omits = stmt.Omits
				}
			}

			expr := clause.NamedExpr{SQL: fmt.Sprintf("%s JOIN (?) AS ?", joinType), Vars: []interface{}{jt.Subquery, clause.Table{Name: j.Alias}}}

			if j.On != nil {
				expr.SQL += " ON ?"
				expr.Vars = append(expr.Vars, clause.AndConditions{Exprs: j.On.Exprs})
			}

			j.Expression = expr
		}

		db.Statement.Joins = append(db.Statement.Joins, j)
		sort.Slice(db.Statement.Joins, func(i, j int) bool {
			return db.Statement.Joins[i].Name < db.Statement.Joins[j].Name
		})
		return db
	})
}

// Select records field selection or a trusted SQL expression with bound args.
// Explicitly selected fields can include zero values in Updates; Omit and field
// permissions still apply.
func (c chainG[T]) Select(query string, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Select(query, args...)
	})
}

// Omit records fields or associations to exclude from subsequent queries or writes. It
// replaces the omit list; see DB.Omit for association wildcard behavior.
func (c chainG[T]) Omit(columns ...string) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Omit(columns...)
	})
}

// MapColumns records a mapping from returned column names to model fields for scanning.
// It retains the map; do not mutate it during execution.
func (c chainG[T]) MapColumns(m map[string]string) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.MapColumns(m)
	})
}

// Set collects column and association assignments for Update on the current query
// chain. It does not execute SQL and does not expose Create after chaining. Association
// operations run before owner assignments; use an explicit transaction for atomicity
// across them.
//
// Current many-to-many association unlink, delete, and update paths rebuild the owner
// EXISTS query without the chain's owner filters. They can therefore affect links or
// related rows belonging to other owners; Association.Conditions filters related
// records, not owners. Use explicit owner-key-qualified operations when owner scoping
// is required. An enclosing transaction provides atomicity but does not restore those
// filters.
func (c chainG[T]) Set(assignments ...clause.Assigner) SetUpdateOnlyInterface[T] {
	return c.processSet(assignments...)
}

// Distinct records DISTINCT and, when supplied, selected fields. Use a trusted field
// selection; SQL expressions are not validated.
func (c chainG[T]) Distinct(args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Distinct(args...)
	})
}

// Group records a GROUP BY column or trusted expression. Pair it with an aggregate
// Select and optional Having predicate.
func (c chainG[T]) Group(name string) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Group(name)
	})
}

// Having records post-group predicates using DB.Having rules. Bind values separately
// from the trusted expression text.
func (c chainG[T]) Having(query interface{}, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Having(query, args...)
	})
}

// Order records sorting using a trusted SQL string or clause.OrderBy/OrderByColumn.
// Validate dynamic identifiers and directions before constructing them.
func (c chainG[T]) Order(value interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Order(value)
	})
}

// Preload records a model association path and an optional callback customizing its
// separate query. Callback errors are recorded on the operation. Nested paths are
// supported; filters affect associated rows rather than parent rows. Keep relationship
// keys in the selected columns.
func (c chainG[T]) Preload(association string, query func(db PreloadBuilder) error) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Preload(association, func(tx *DB) *DB {
			q := preloadBuilder{db: tx.getInstance()}
			if query != nil {
				if err := query(&q); err != nil {
					db.AddError(err)
				}
			}

			relation, ok := db.Statement.Schema.Relationships.Relations[association]
			if !ok {
				if preloadFields := strings.Split(association, "."); len(preloadFields) > 1 {
					relationships := &db.Statement.Schema.Relationships
					for _, field := range preloadFields {
						var ok bool
						relation, ok = relationships.Relations[field]
						if ok {
							relationships = &relation.FieldSchema.Relationships
						} else {
							db.AddError(fmt.Errorf("relation %s not found", association))
							return nil
						}
					}
				} else {
					db.AddError(fmt.Errorf("relation %s not found", association))
					return nil
				}
			}

			if q.limitPerRecord > 0 {
				if relation.JoinTable != nil {
					tx.AddError(fmt.Errorf("many2many relation %s don't support LimitPerRecord", association))
					return tx
				}

				refColumns := []clause.Column{}
				for _, rel := range relation.References {
					if rel.OwnPrimaryKey {
						refColumns = append(refColumns, clause.Column{Name: rel.ForeignKey.DBName})
					}
				}

				if len(refColumns) != 0 {
					selectExpr := clause.CommaExpression{}
					for _, column := range q.db.Statement.Selects {
						selectExpr.Exprs = append(selectExpr.Exprs, clause.Expr{SQL: "?", Vars: []interface{}{clause.Column{Name: column}}})
					}

					if len(selectExpr.Exprs) == 0 {
						selectExpr.Exprs = []clause.Expression{clause.Expr{SQL: "*", Vars: []interface{}{}}}
					}

					partitionBy := clause.CommaExpression{}
					for _, column := range refColumns {
						partitionBy.Exprs = append(partitionBy.Exprs, clause.Expr{SQL: "?", Vars: []interface{}{clause.Column{Name: column.Name}}})
					}

					rnnColumn := clause.Column{Name: "gorm_preload_rnn"}
					sql := "ROW_NUMBER() OVER (PARTITION BY ? ?)"
					vars := []interface{}{partitionBy}
					if orderBy, ok := q.db.Statement.Clauses["ORDER BY"]; ok {
						vars = append(vars, orderBy)
					} else {
						vars = append(vars, clause.Clause{Name: "ORDER BY", Expression: clause.OrderBy{
							Columns: []clause.OrderByColumn{{Column: clause.PrimaryColumn, Desc: true}},
						}})
					}
					vars = append(vars, rnnColumn)

					selectExpr.Exprs = append(selectExpr.Exprs, clause.Expr{SQL: sql + " AS ?", Vars: vars})

					q.db.Clauses(clause.Select{Expression: selectExpr})

					return q.db.Session(&Session{NewDB: true}).Unscoped().Table("(?) t", q.db).Where("? <= ?", rnnColumn, q.limitPerRecord)
				}
			}

			return q.db
		})
	})
}

// Delete removes rows for model T matching the chain and returns the affected-row count
// and error. It starts from a zero-valued model, so normally provide Where conditions.
// A soft-delete-capable field type such as gorm.DeletedAt enables soft deletion; the
// field name alone does not. The missing-WHERE guard still applies.
func (c chainG[T]) Delete(ctx context.Context) (rowsAffected int, err error) {
	r := new(T)
	res := c.g.apply(ctx).Delete(r)
	return int(res.RowsAffected), res.Error
}

// Update writes one column for model T matching the chain and returns affected rows and
// an error. Model hooks and update timestamps apply. Supply conditions unless a global
// update is intended and explicitly enabled.
//
// For struct-valued T, this path uses a non-addressable model value. A pointer-receiver
// BeforeSave, BeforeUpdate, AfterUpdate, or AfterSave hook can cause ErrInvalidValue
// rather than being invoked. Use the traditional Model(&model).Update/Updates path when
// addressable update hooks are required.
func (c chainG[T]) Update(ctx context.Context, name string, value any) (rowsAffected int, err error) {
	var r T
	res := c.g.apply(ctx).Model(r).Update(name, value)
	return int(res.RowsAffected), res.Error
}

// Updates writes fields from t using DB.Updates rules and returns affected rows and an
// error. Struct zero values are omitted unless selected; map-valued T includes zero
// values. Conditions or a non-zero model key must restrict the write.
//
// For struct-valued T, this path uses a non-addressable model value. A pointer-receiver
// BeforeSave, BeforeUpdate, AfterUpdate, or AfterSave hook can cause ErrInvalidValue
// rather than being invoked. Use the traditional Model(&model).Update/Updates path when
// addressable update hooks are required.
func (c chainG[T]) Updates(ctx context.Context, t T) (rowsAffected int, err error) {
	res := c.g.apply(ctx).Updates(t)
	return int(res.RowsAffected), res.Error
}

// Count returns an aggregate for model T and the chain. Use "*" for all rows or a
// single column name to count non-NULL values. Distinct counts unique column values;
// Group returns the number of groups. The string is passed through Select and DB.Count:
// it is not an arbitrary expression argument. An explicit trusted count(...) selection
// is retained, while unsupported selections can fall back to count(*).
func (c chainG[T]) Count(ctx context.Context, column string) (result int64, err error) {
	var r T
	err = c.g.apply(ctx).Model(r).Select(column).Count(&result).Error
	return
}

// Build renders this query as a subquery into a *Statement builder, rebinding
// placeholders into its variable sequence. It uses DryRun query callbacks and emits
// nothing for other builder implementations; it does not fetch rows.
func (c chainG[T]) Build(builder clause.Builder) {
	subdb := c.getInstance()
	subdb.Logger = logger.Discard
	subdb.DryRun = true

	if stmt, ok := builder.(*Statement); ok {
		if subdb.Statement.SQL.Len() > 0 {
			var (
				vars = subdb.Statement.Vars
				sql  = subdb.Statement.SQL.String()
			)

			subdb.Statement.Vars = make([]interface{}, 0, len(vars))
			for _, vv := range vars {
				subdb.Statement.Vars = append(subdb.Statement.Vars, vv)
				bindvar := strings.Builder{}
				subdb.BindVarTo(&bindvar, subdb.Statement, vv)
				sql = strings.Replace(sql, bindvar.String(), "?", 1)
			}

			subdb.Statement.SQL.Reset()
			subdb.Statement.Vars = stmt.Vars
			if strings.Contains(sql, "@") {
				clause.NamedExpr{SQL: sql, Vars: vars}.Build(subdb.Statement)
			} else {
				clause.Expr{SQL: sql, Vars: vars}.Build(subdb.Statement)
			}
		} else {
			subdb.Statement.Vars = append(stmt.Vars, subdb.Statement.Vars...)
			subdb.callbacks.Query().Execute(subdb)
		}

		builder.WriteString(subdb.Statement.SQL.String())
		stmt.Vars = subdb.Statement.Vars
	}
}

// execG runs a recorded generic chain through the traditional finishers and converts
// destinations and error returns to the typed API.
type execG[T any] struct {
	// g holds the source DB and deferred operations applied for each execution.
	g *g[T]
}

// First returns one T using ascending primary-key ordering, or ErrRecordNotFound when
// no row matches. Check err before using the returned value. Existing chain ordering
// and conditions still participate.
func (g execG[T]) First(ctx context.Context) (T, error) {
	var r T
	err := g.g.apply(ctx).First(&r).Error
	return r, err
}

// Scan executes the typed model query into a writable destination such as a projection
// struct or slice. This generic method uses the normal Find callback pipeline, unlike
// traditional DB.Scan, which uses row callbacks. It returns the execution error
// directly.
func (g execG[T]) Scan(ctx context.Context, result interface{}) error {
	var r T
	err := g.g.apply(ctx).Model(r).Find(result).Error
	return err
}

// Last returns one T using descending primary-key ordering, or ErrRecordNotFound on a
// miss. Primary-key order is not necessarily creation-time order.
func (g execG[T]) Last(ctx context.Context) (T, error) {
	var r T
	err := g.g.apply(ctx).Last(&r).Error
	return r, err
}

// Take returns one T without adding an order; an existing Order is retained. It returns
// ErrRecordNotFound when nothing matches.
func (g execG[T]) Take(ctx context.Context) (T, error) {
	var r T
	err := g.g.apply(ctx).Take(&r).Error
	return r, err
}

// Find returns matching rows as []T and an error. An empty result normally has no
// error; len(result) reports whether anything matched.
func (g execG[T]) Find(ctx context.Context) ([]T, error) {
	var r []T
	err := g.g.apply(ctx).Find(&r).Error
	return r, err
}

// FindInBatches calls fc for successive non-empty model batches; numbering starts at 1
// and batchSize must be positive. Include the prioritized primary key for continuation
// and do not change it during fc. The batch slice may be reused: copy data that must
// outlive the callback. Query or callback errors stop iteration; no snapshot
// transaction is created automatically.
//
// Traversal uses an ascending prioritized-primary-key cursor. Avoid conflicting Order
// clauses, including descending key order, which can skip or repeat records; see
// DB.FindInBatches.
func (g execG[T]) FindInBatches(ctx context.Context, batchSize int, fc func(data []T, batch int) error) error {
	var data []T
	return g.g.apply(ctx).FindInBatches(&data, batchSize, func(tx *DB, batch int) error {
		return fc(data, batch)
	}).Error
}

// Row returns a *sql.Row for manual Scan; query errors and sql.ErrNoRows are reported
// by Scan. DryRun or construction failure can leave no row. This path does not perform
// ordinary model preloading or query hooks.
func (g execG[T]) Row(ctx context.Context) *sql.Row {
	var r T
	return g.g.apply(ctx).Model(r).Row()
}

// Rows returns an owned result set or an error. On success, close the rows, iterate
// with Next/Scan, and check Err after iteration. DryRun is unsupported; this path does
// not perform normal query preloading or hooks.
func (g execG[T]) Rows(ctx context.Context) (*sql.Rows, error) {
	var r T
	return g.g.apply(ctx).Model(r).Rows()
}

// processSet separates ordinary column assignments from AssociationAssigner operations.
// It records their values without executing SQL; association operations later run
// before owner assignments.
func (c chainG[T]) processSet(items ...clause.Assigner) setCreateOrUpdateG[T] {
	var (
		assigns  []clause.Assignment
		assocOps []clause.Association
	)

	for _, item := range items {
		// Check if it's an AssociationAssigner
		if assocAssigner, ok := item.(clause.AssociationAssigner); ok {
			assocOps = append(assocOps, assocAssigner.AssociationAssignments()...)
		} else {
			assigns = append(assigns, item.Assignments()...)
		}
	}

	return setCreateOrUpdateG[T]{
		c:        c,
		assigns:  assigns,
		assocOps: assocOps,
	}
}

// setCreateOrUpdateG stores a target chain, column assignments, and association
// operations. Execution processes associations before the owner write, so an enclosing
// transaction is required for all-or-nothing behavior across the complete set.
type setCreateOrUpdateG[T any] struct {
	// c supplies the owner model, conditions, and execution connection.
	c chainG[T]
	// assigns contains owner-column assignments applied after association operations.
	assigns []clause.Assignment
	// assocOps contains related-record operations executed sequentially before owner
	// assignments.
	assocOps []clause.Association
}

// Update executes association operations first and then owner-column assignments,
// stopping on the first error. The returned row count covers the owner update, not all
// association changes; association-only work returns zero. Separate operations are not
// automatically wrapped in one transaction.
//
// Current many-to-many association unlink, delete, and update paths rebuild the owner
// EXISTS query without the chain's owner filters. They can therefore affect links or
// related rows belonging to other owners; Association.Conditions filters related
// records, not owners. Use explicit owner-key-qualified operations when owner scoping
// is required. An enclosing transaction provides atomicity but does not restore those
// filters.
//
// For struct-valued T, the owner update uses a non-addressable model value;
// pointer-receiver update/save hooks can produce ErrInvalidValue. Use traditional
// Model(&model) updates when those hooks are required.
func (s setCreateOrUpdateG[T]) Update(ctx context.Context) (rowsAffected int, err error) {
	// Execute association operations
	for _, assocOp := range s.assocOps {
		if err := s.executeAssociationOperation(ctx, assocOp); err != nil {
			return 0, err
		}
	}

	// Execute assignment operations
	if len(s.assigns) > 0 {
		var r T
		res := s.c.g.apply(ctx).Model(r).Clauses(clause.Set(s.assigns)).Updates(map[string]interface{}{})
		return int(res.RowsAffected), res.Error
	}

	return 0, nil
}

// Create executes association operations and then inserts a map built from owner-column
// assignments. Map-create rules apply: no model hook calls or model key backfill.
// Association-only work performs no owner insert. Use a transaction when all operations
// must succeed together.
func (s setCreateOrUpdateG[T]) Create(ctx context.Context) error {
	// Execute association operations
	for _, assocOp := range s.assocOps {
		if err := s.executeAssociationOperation(ctx, assocOp); err != nil {
			return err
		}
	}

	// Execute assignment operations
	if len(s.assigns) > 0 {
		data := make(map[string]interface{}, len(s.assigns))
		for _, a := range s.assigns {
			data[a.Column.Name] = a.Value
		}
		var r T
		return s.c.g.apply(ctx).Model(r).Create(data).Error
	}

	return nil
}

// executeAssociationOperation resolves the owner model and dispatches create, unlink,
// delete, or update operations. Unknown operation types return an error without
// executing an association handler.
func (s setCreateOrUpdateG[T]) executeAssociationOperation(ctx context.Context, op clause.Association) error {
	var r T
	base := s.c.g.apply(ctx).Model(r)

	switch op.Type {
	case clause.OpCreate:
		return s.handleAssociationCreate(ctx, base, op)
	case clause.OpUnlink, clause.OpDelete, clause.OpUpdate:
		return s.handleAssociation(ctx, base, op)
	default:
		return fmt.Errorf("unknown association operation type: %v", op.Type)
	}
}

// handleAssociationCreate finds owners and appends association values for each.
// Set-based input is converted to a map; otherwise op.Values is passed to
// Association.Append. A failure stops later owners.
func (s setCreateOrUpdateG[T]) handleAssociationCreate(ctx context.Context, base *DB, op clause.Association) error {
	if len(op.Set) > 0 {
		return s.handleAssociationForOwners(base, ctx, func(owner T, assoc *Association) error {
			data := make(map[string]interface{}, len(op.Set))
			for _, a := range op.Set {
				data[a.Column.Name] = a.Value
			}
			return assoc.Append(data)
		}, op.Association)
	}

	return s.handleAssociationForOwners(base, ctx, func(owner T, assoc *Association) error {
		return assoc.Append(op.Values...)
	}, op.Association)
}

// handleAssociationForOwners loads all matching owners, creates a fresh association
// handle for each, and calls handler sequentially. It returns the first query,
// association setup, or handler error and does not create an encompassing transaction.
func (s setCreateOrUpdateG[T]) handleAssociationForOwners(base *DB, ctx context.Context, handler func(owner T, association *Association) error, associationName string) error {
	var owners []T
	if err := base.Find(&owners).Error; err != nil {
		return err
	}

	for _, owner := range owners {
		assoc := base.Session(&Session{NewDB: true, Context: ctx}).Model(&owner).Association(associationName)
		if assoc.Error != nil {
			return assoc.Error
		}

		if err := handler(owner, assoc); err != nil {
			return err
		}
	}
	return nil
}

// handleAssociation maps the operation to relationship-specific SQL. Has-one/has-many
// use owner-key subqueries; belongs-to deletion also clears references in a
// transaction. Many-to-many unlink/delete removes join rows, while update targets
// related rows using correlated EXISTS queries.
//
// The many-to-many owner EXISTS statement starts with NewDB and loses the chain's owner
// predicates. It checks that an owner exists, not that the owner matches the original
// chain. Related Conditions remain applicable, but they cannot supply an owner-scope
// guarantee.
func (s setCreateOrUpdateG[T]) handleAssociation(ctx context.Context, base *DB, op clause.Association) error {
	assoc := base.Association(op.Association)
	if assoc.Error != nil {
		return assoc.Error
	}

	var (
		rel            = assoc.Relationship
		assocModel     = reflect.New(rel.FieldSchema.ModelType).Interface()
		fkNil          = map[string]any{}
		setMap         = make(map[string]any, len(op.Set))
		ownerPKNames   []string
		ownerFKNames   []string
		primaryColumns []any
		foreignColumns []any
	)

	for _, a := range op.Set {
		setMap[a.Column.Name] = a.Value
	}

	for _, ref := range rel.References {
		fkNil[ref.ForeignKey.DBName] = nil

		if ref.OwnPrimaryKey && ref.PrimaryKey != nil {
			ownerPKNames = append(ownerPKNames, ref.PrimaryKey.DBName)
			primaryColumns = append(primaryColumns, clause.Column{Name: ref.PrimaryKey.DBName})
			foreignColumns = append(foreignColumns, clause.Column{Name: ref.ForeignKey.DBName})
		} else if !ref.OwnPrimaryKey && ref.PrimaryKey != nil {
			ownerFKNames = append(ownerFKNames, ref.ForeignKey.DBName)
			primaryColumns = append(primaryColumns, clause.Column{Name: ref.PrimaryKey.DBName})
		}
	}

	assocDB := s.c.g.db.Session(&Session{NewDB: true, Context: ctx}).Model(assocModel).Where(op.Conditions)

	switch rel.Type {
	case schema.HasOne, schema.HasMany:
		assocDB = assocDB.Where("? IN (?)", foreignColumns, base.Select(ownerPKNames))
		switch op.Type {
		case clause.OpUnlink:
			return assocDB.Updates(fkNil).Error
		case clause.OpDelete:
			return assocDB.Delete(assocModel).Error
		case clause.OpUpdate:
			return assocDB.Updates(setMap).Error
		}
	case schema.BelongsTo:
		switch op.Type {
		case clause.OpDelete:
			return base.Transaction(func(tx *DB) error {
				assocDB.Statement.ConnPool = tx.Statement.ConnPool
				base.Statement.ConnPool = tx.Statement.ConnPool

				if err := assocDB.Where("? IN (?)", primaryColumns, base.Select(ownerFKNames)).Delete(assocModel).Error; err != nil {
					return err
				}
				return base.Updates(fkNil).Error
			})
		case clause.OpUnlink:
			return base.Updates(fkNil).Error
		case clause.OpUpdate:
			return assocDB.Where("? IN (?)", primaryColumns, base.Select(ownerFKNames)).Updates(setMap).Error
		}
	case schema.Many2Many:
		joinModel := reflect.New(rel.JoinTable.ModelType).Interface()
		joinDB := base.Session(&Session{NewDB: true, Context: ctx}).Model(joinModel)

		// EXISTS owners: owners.pk = join.owner_fk for all owner refs
		ownersExists := base.Session(&Session{NewDB: true, Context: ctx}).Table(rel.Schema.Table).Select("1")
		for _, ref := range rel.References {
			if ref.OwnPrimaryKey && ref.PrimaryKey != nil {
				ownersExists = ownersExists.Where(clause.Eq{
					Column: clause.Column{Table: rel.Schema.Table, Name: ref.PrimaryKey.DBName},
					Value:  clause.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
				})
			}
		}

		// EXISTS related: related.pk = join.rel_fk for all related refs, plus optional conditions
		relatedExists := base.Session(&Session{NewDB: true, Context: ctx}).Table(rel.FieldSchema.Table).Select("1")
		for _, ref := range rel.References {
			if !ref.OwnPrimaryKey && ref.PrimaryKey != nil {
				relatedExists = relatedExists.Where(clause.Eq{
					Column: clause.Column{Table: rel.FieldSchema.Table, Name: ref.PrimaryKey.DBName},
					Value:  clause.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
				})
			}
		}
		relatedExists = relatedExists.Where(op.Conditions)

		switch op.Type {
		case clause.OpUnlink, clause.OpDelete:
			joinDB = joinDB.Where("EXISTS (?)", ownersExists)
			if len(op.Conditions) > 0 {
				joinDB = joinDB.Where("EXISTS (?)", relatedExists)
			}
			return joinDB.Delete(nil).Error
		case clause.OpUpdate:
			// Update related table rows that have join rows matching owners
			relatedDB := base.Session(&Session{NewDB: true, Context: ctx}).Table(rel.FieldSchema.Table).Where(op.Conditions)

			// correlated join subquery: join.rel_fk = related.pk AND EXISTS owners
			joinSub := base.Session(&Session{NewDB: true, Context: ctx}).Table(rel.JoinTable.Table).Select("1")
			for _, ref := range rel.References {
				if !ref.OwnPrimaryKey && ref.PrimaryKey != nil {
					joinSub = joinSub.Where(clause.Eq{
						Column: clause.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
						Value:  clause.Column{Table: rel.FieldSchema.Table, Name: ref.PrimaryKey.DBName},
					})
				}
			}
			joinSub = joinSub.Where("EXISTS (?)", ownersExists)
			return relatedDB.Where("EXISTS (?)", joinSub).Updates(setMap).Error
		}
	}
	return errors.New("unsupported relationship")
}
