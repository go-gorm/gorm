### Connecting To A Database

```go
import (
    "github.com/jinzhu/gorm"
    _ "github.com/lib/pq"
    _ "github.com/go-sql-driver/mysql"
    _ "github.com/mattn/go-sqlite3"
)

func init() {
  db, err := gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
  // db, err := gorm.Open("mysql", "user:password@/dbname?charset=utf8&parseTime=True&loc=Local")
  // db, err := gorm.Open("sqlite3", "/tmp/gorm.db")

  // Use existing database connection
  dbSql, err := sql.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
  db, err := gorm.Open("postgres", dbSql)
}
```

```go
// Get database connection handle [*sql.DB](http://golang.org/pkg/database/sql/#DB)
db.DB()

// Then you could invoke `*sql.DB`'s functions with it
db.DB().Ping()
db.DB().SetMaxIdleConns(10)
db.DB().SetMaxOpenConns(100)

// Disable table name's pluralization
db.SingularTable(true)
```
