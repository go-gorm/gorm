// Package callbacks implements GORM's default create, query, update, delete,
// row, and raw processors. Dialectors install them with
// RegisterDefaultCallbacks; applications normally use gorm.DB methods instead
// of calling individual callbacks. Each callback expects a statement prepared
// by the processor, including its schema, destination, context, and clause map.
//
// # Lifecycle and extension points
//
// The default create sequence is transaction begin, BeforeSave/BeforeCreate,
// belongs-to association saves, INSERT, remaining association saves,
// AfterCreate/AfterSave, and commit or rollback. Update follows the analogous
// sequence with BeforeUpdate/AfterUpdate. Delete runs BeforeDelete, explicitly
// selected association deletion, owner deletion, AfterDelete, and transaction
// completion. Schema clauses can turn owner deletion into a soft-delete UPDATE.
// Queries execute SELECT, load preloads, then call AfterFind on found models.
// Row and raw processors do not run those model hooks or preloads.
//
// A model hook receives a fresh statement on the operation's connection or
// transaction. Its queries do not inherit the outer WHERE conditions. Hook
// errors are recorded on the operation and cause an owned default write
// transaction to roll back. After hooks run before commit; use application
// code after a successful Transaction return for actions requiring commit
// success. SkipHooks suppresses model hooks, not all registered callbacks,
// automatic association work, or schema clauses. DryRun prevents SQL execution
// by the SQL callbacks but does not make the entire callback chain side-effect
// free: hooks and conversion of defaults/timestamps can still mutate values.
//
// Register extensions using db.Callback().Create(), Query(), Update(), Delete(),
// Row(), or Raw(), ordering them relative to names such as "gorm:create" and
// "gorm:query". Registries are shared by sessions; install extensions during
// initialization. Replacing one SQL callback does not automatically replace
// the hooks, association work, or transaction callbacks around it. Report
// failures with db.AddError and check db.Error before executing dependent work.
//
// # Values, associations, and query results
//
// Create conversion respects field permissions and Select/Omit, fills model
// defaults and timestamps, and leaves database-generated values to SQL defaults
// or RETURNING/LastInsertId handling. Map input retains explicit zero values
// and does not receive the same model-default filling. Struct updates normally
// omit zero values; map updates and explicitly selected fields can write them.
// The missing-WHERE guard is structural: it rejects an absent condition, not a
// caller-supplied condition that happens to match every row.
//
// Association saves use conflict handling and share the current connection or
// transaction. FullSaveAssociations expands updates to related records; it is
// not a replacement operation that deletes omitted related records. Object
// identity tracking breaks recursive save cycles within an operation, not
// duplicate requests. Preload uses separate queries and identity maps, so
// projections must retain the relationship keys required to attach results.
// Joined associations are traversed without reloading that relationship.
// Ordinary preload limits apply to the whole related query, not per parent.
// Use an explicit transaction when a consistent parent/association snapshot is
// required, subject to the database's isolation semantics.
//
// SQL callbacks bind values through the statement and respect its context.
// Query and RETURNING callbacks close the rows they consume; RowQuery transfers
// row ownership to the caller. Raw SQL fragments and SQL expressions remain
// trusted input, even when their accompanying values are bound parameters.
//
// For application examples, see https://gorm.io/docs/hooks.html,
// https://gorm.io/docs/associations.html, and https://gorm.io/docs/preload.html.
package callbacks
