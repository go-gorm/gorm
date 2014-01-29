# Gorm Development

## Architecture

The most notable component of Gorm is `gorm.DB`, which hold database connection. It could be initialized like this:

    db, err := gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")

Gorm has chainable API, `gorm.DB` is the bridge of chains, it save related information and pass it to the next chain.

Lets use below code to explain how it works:

    db.Where("name = ?", "jinzhu").Find(&users)

    // equivalent code
    newdb := db.Where("name =?", "jinzhu")
    newdb.Find(&user)

`newdb` is `db`'s clone, in addition, it contains search conditions from the `Where` method.
`Find` is a query method, it creates a `Scope` instance, and pass it as argument to query callbacks.

There are four kinds of callbacks corresponds to sql's CURD: create callbacks, update callbacks, query callbacks, delete callbacks.

## Callbacks

### Register a new callback

    func updateCreated(scope *Scope) {
        if scope.HasColumn("Created") {
            scope.SetColumn("Created", time.Now())
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

### Callback API

Gorm is powered by callbacks, so you could refer below links to learn how to write callbacks

[Create callbacks](https://github.com/jinzhu/gorm/blob/master/callback_create.go)

[Update callbacks](https://github.com/jinzhu/gorm/blob/master/callback_update.go)

[Query callbacks](https://github.com/jinzhu/gorm/blob/master/callback_create.go)

[Delete callbacks](https://github.com/jinzhu/gorm/blob/master/callback_delete.go)

View [https://github.com/jinzhu/gorm/blob/master/scope.go](https://github.com/jinzhu/gorm/blob/master/scope.go) for all available API
