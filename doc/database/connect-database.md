# Database Connection

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
