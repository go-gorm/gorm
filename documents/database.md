# Database

<!-- toc -->

## Connecting to a database

In order to connect to a database, you need to import the database's driver first. For example:

```go
import _ "github.com/go-sql-driver/mysql"
```

GORM has wrapped some drivers, for easier to remember the import path, so you could import the mysql driver with

```go
import _ "github.com/jinzhu/gorm/dialects/mysql"
// import _ "github.com/jinzhu/gorm/dialects/postgres"
// import _ "github.com/jinzhu/gorm/dialects/sqlite"
// import _ "github.com/jinzhu/gorm/dialects/mssql"
```

#### MySQL

**NOTE:** In order to handle `time.Time`, you need to include `parseTime` as a parameter. ([More supported parameters](https://github.com/go-sql-driver/mysql#parameters))

```go
import (
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/mysql"
)

func main() {
  db, err := gorm.Open("mysql", "user:password@/dbname?charset=utf8&parseTime=True&loc=Local")
}
```

#### PostgreSQL

```go
import (
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/postgres"
)

func main() {
  db, err := gorm.Open("postgres", "host=myhost user=gorm dbname=gorm sslmode=disable password=mypassword")
}
```

#### Sqlite3

```go
import (
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/sqlite"
)

func main() {
  db, err := gorm.Open("sqlite3", "/tmp/gorm.db")
}
```

#### Write Dialect for unsupported databases

GORM officially supports the above databases, but you could write a dialect for unsupported databases.

To write your own dialect, refer to: [https://github.com/jinzhu/gorm/blob/master/dialect.go](https://github.com/jinzhu/gorm/blob/master/dialect.go)

## Migration

### Auto Migration

Automatically migrate your schema, to keep your schema update to date.

**WARNING:** AutoMigrate will **ONLY** create tables, missing columns and missing indexes, and **WON'T** change existing column's type or delete unused columns to protect your data.

```go
db.AutoMigrate(&User{})

db.AutoMigrate(&User{}, &Product{}, &Order{})

// Add table suffix when create tables
db.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(&User{})
```

### Has Table

```go
// Check model `User`'s table exists or not
db.HasTable(&User{})

// Check table `users` exists or not
db.HasTable("users")
```

### Create Table

```go
// Create table for model `User`
db.CreateTable(&User{})

// will append "ENGINE=InnoDB" to the SQL statement when creating table `users`
db.Set("gorm:table_options", "ENGINE=InnoDB").CreateTable(&User{})
```

### Drop table

```go
// Drop model `User`'s table
db.DropTable(&User{})

// Drop table `users
db.DropTable("users")

// Drop model's `User`'s table and table `products`
db.DropTableIfExists(&User{}, "products")
```

### ModifyColumn

Modify column's type to given value

```go
// change column description's data type to `text` for model `User`
db.Model(&User{}).ModifyColumn("description", "text")
```

### DropColumn

```go
// Drop column description from model `User`
db.Model(&User{}).DropColumn("description")
```

### Add Foreign Key

```go
// Add foreign key
// 1st param : foreignkey field
// 2nd param : destination table(id)
// 3rd param : ONDELETE
// 4th param : ONUPDATE
db.Model(&User{}).AddForeignKey("city_id", "cities(id)", "RESTRICT", "RESTRICT")
```

### Indexes

```go
// Add index for columns `name` with given name `idx_user_name`
db.Model(&User{}).AddIndex("idx_user_name", "name")

// Add index for columns `name`, `age` with given name `idx_user_name_age`
db.Model(&User{}).AddIndex("idx_user_name_age", "name", "age")

// Add unique index
db.Model(&User{}).AddUniqueIndex("idx_user_name", "name")

// Add unique index for multiple columns
db.Model(&User{}).AddUniqueIndex("idx_user_name_age", "name", "age")

// Remove index
db.Model(&User{}).RemoveIndex("idx_user_name")
```
