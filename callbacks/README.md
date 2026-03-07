# callbacks

This package registers the default lifecycle callbacks for each GORM database operation.

## How it works

Every GORM operation (Create, Query, Update, Delete) runs a chain of named callbacks in order. Database dialects call `RegisterDefaultCallbacks` during initialization and can pass a `Config` to override which SQL clauses are used.

## Callback chains

| Operation | Order |
|-----------|-------|
| **Create** | `begin_transaction` → `before_create` → `save_before_associations` → `create` → `save_after_associations` → `after_create` → `commit_or_rollback_transaction` |
| **Query**  | `query` → `preload` → `after_query` |
| **Update** | `begin_transaction` → `setup_reflect_value` → `before_update` → `save_before_associations` → `update` → `save_after_associations` → `after_update` → `commit_or_rollback_transaction` |
| **Delete** | `begin_transaction` → `before_delete` → `delete_before_associations` → `delete` → `after_delete` → `commit_or_rollback_transaction` |
| **Row**    | `row` |
| **Raw**    | `raw` |

## Key files

| File | Description |
|------|-------------|
| `callbacks.go` | `RegisterDefaultCallbacks` — entry point called by dialects |
| `create.go` | Insert logic and `BeforeCreate`/`AfterCreate` hooks |
| `query.go` | Select logic and `AfterQuery` hook |
| `update.go` | Update logic and `BeforeUpdate`/`AfterUpdate` hooks |
| `delete.go` | Delete logic and `BeforeDelete`/`AfterDelete` hooks |
| `associations.go` | Save/delete associated records |
| `preload.go` | Eager-load associations after query |
| `transaction.go` | Begin and commit/rollback transaction wrappers |
| `callmethod.go` | Reflection helpers to invoke model hook methods |
| `helper.go` | Shared utilities used across callbacks |

## Customizing callbacks

```go
// Add a callback before an existing one
db.Callback().Create().Before("gorm:create").Register("my_plugin:audit", auditFn)

// Replace a callback
db.Callback().Delete().Replace("gorm:delete", softDeleteFn)

// Remove a callback
db.Callback().Update().Remove("gorm:before_update")
```

See [GORM hooks documentation](https://gorm.io/docs/hooks.html) for the full list of model hook methods (`BeforeCreate`, `AfterSave`, etc.).
