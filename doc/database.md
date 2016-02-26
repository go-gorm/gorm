# Database

<!-- toc -->

## Connecting to a database

#### MySQL

**NOTE** don't forgot params `parseTime` to handle data type `time.Time`, [more support parameters](https://github.com/go-sql-driver/mysql#parameters)

```go
import (
    "github.com/jinzhu/gorm"
    _ "github.com/go-sql-driver/mysql"
)
func main() {
  db, err := gorm.Open("mysql", "user:password@/dbname?charset=utf8&parseTime=True&loc=Local")
}
```

#### PostgreSQL

```go
import (
    "github.com/jinzhu/gorm"
    _ "github.com/lib/pq"
)
func main() {
  db, err := gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
}
```

#### Sqlite3

```go
import (
    "github.com/jinzhu/gorm"
    _ "github.com/mattn/go-sqlite3"
)
func main() {
  db, err := gorm.Open("sqlite3", "/tmp/gorm.db")
}
```

#### Write Dialect for unsupported databases

GORM officially support above databases, for unsupported databaes, you could write a dialect for that.

Refer: https://github.com/jinzhu/gorm/blob/master/dialect.go


## Generic database object *sql.DB

[*sql.DB](http://golang.org/pkg/database/sql/#DB)

```go
// Get generic database object *sql.DB to use its functions
db.DB()

// Connection Pool
db.DB().SetMaxIdleConns(10)
db.DB().SetMaxOpenConns(100)

  // Ping
db.DB().Ping()
```

## Migration

<!-- toc -->

### Auto Migration

Automatically migrate your schema, to keep your schema update to date

**WARNING** AutoMigrate will ONLY create tables, columns and indexes if doesn't exist,
WON'T change existing column's type or delete unused columns to protect your data

```go
db.AutoMigrate(&User{})

db.AutoMigrate(&User{}, &Product{}, &Order{})

// Add table suffix when create tables
db.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(&User{})
```

### Has Table

```go
// Check if model `User`'s table has been created or not
db.HasTable(&User{})

// Check table `users` exists or not
db.HasTable("users")
```

### Create Table

```go
db.CreateTable(&User{})

db.Set("gorm:table_options", "ENGINE=InnoDB").CreateTable(&User{})
// will append "ENGINE=InnoDB" to the SQL statement when creating table `users`
```

### Drop table

```go
db.DropTable(&User{})
```

### ModifyColumn

Change column's type

```go
// change column description's data type to `text` for model `User`'s table
db.Model(&User{}).ModifyColumn("description", "text")
```

### DropColumn

```go
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
// Add index
db.Model(&User{}).AddIndex("idx_user_name", "name")

// Multiple column index
db.Model(&User{}).AddIndex("idx_user_name_age", "name", "age")

// Add unique index
db.Model(&User{}).AddUniqueIndex("idx_user_name", "name")

// Multiple column unique index
db.Model(&User{}).AddUniqueIndex("idx_user_name_age", "name", "age")

// Remove index
db.Model(&User{}).RemoveIndex("idx_user_name")
```
