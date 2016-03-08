# Development

<!-- toc -->

## Architecture

Gorm use chainable API, `*gorm.DB` is the bridge of chains, for each chain API, it will create a new relation.

```go
db, err := gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")

// create a new relation
db = db.Where("name = ?", "jinzhu")

// create another new relation
db = db.Where("age = ?", 20)
```

When we start to perform any operations, GROM will create a new `*gorm.Scope` instance based on current `*gorm.DB`

```go
// perform a querying operation
db.First(&user)
```

And based on current operation's type, it will call registered `creating`, `updating`, `querying`, `deleting` or `row_querying` callbacks to run the operation.

For above example, will call `querying` callbacks, refer [Querying Callbacks](callbacks.html#querying-an-object)

## Write Plugins

GORM itself is powered by `Callbacks`, so you could fully customize GORM as you want

### Register a new callback

    func updateCreated(scope *Scope) {
        if scope.HasColumn("Created") {
            scope.SetColumn("Created", NowFunc())
        }
    }

    db.Callback().Create().Register("update_created_at", updateCreated)
    // register a callback for Create process

### Delete an existing callback

    db.Callback().Create().Remove("gorm:create")
    // delete callback `gorm:create` from Create callbacks

### Replace an existing callback

    db.Callback().Create().Replace("gorm:create", newCreateFunction)
    // replace callback `gorm:create` with new function `newCreateFunction` for Create process

### Register callback orders

    db.Callback().Create().Before("gorm:create").Register("update_created_at", updateCreated)
    db.Callback().Create().After("gorm:create").Register("update_created_at", updateCreated)
    db.Callback().Query().After("gorm:query").Register("my_plugin:after_query", afterQuery)
    db.Callback().Delete().After("gorm:delete").Register("my_plugin:after_delete", afterDelete)
    db.Callback().Update().Before("gorm:update").Register("my_plugin:before_update", beforeUpdate)
    db.Callback().Create().Before("gorm:create").After("gorm:before_create").Register("my_plugin:before_create", beforeCreate)

### Pre-Defined Callbacks

GORM has defiend callbacks to perform its CRUD operations, check them out before start write your plugins

- [Create callbacks](https://github.com/jinzhu/gorm/blob/master/callback_create.go)

- [Update callbacks](https://github.com/jinzhu/gorm/blob/master/callback_update.go)

- [Query callbacks](https://github.com/jinzhu/gorm/blob/master/callback_query.go)

- [Delete callbacks](https://github.com/jinzhu/gorm/blob/master/callback_delete.go)

- Row Query callbacks

Row Query callbacks will be called when run `Row` or `Rows`, by default there is no registered callbacks for it, you could register a new one like:

```go
func updateTableName(scope *gorm.Scope) {
  scope.Search.Table(scope.TableName() + "_draft") // append `_draft` to table name
}

db.Callback().RowQuery().Register("publish:update_table_name", updateTableName)
```

View [https://godoc.org/github.com/jinzhu/gorm](https://godoc.org/github.com/jinzhu/gorm) to view all available API
