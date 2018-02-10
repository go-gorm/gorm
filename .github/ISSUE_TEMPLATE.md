Your issue may already be reported! Please search on the [issue track](https://github.com/jinzhu/gorm/issues) before creating one.

### What version of Go are you using (`go version`)?


### Which database and its version are you using?


### Please provide a complete runnable program to reproduce your issue. **IMPORTANT**

Need to runnable with [GORM's docker compose config](https://github.com/jinzhu/gorm/blob/master/docker-compose.yml) or please provides your config.

```go
package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var db *gorm.DB

func init() {
	var err error
	db, err = gorm.Open("sqlite3", "test.db")
	// db, err = gorm.Open("postgres", "user=gorm password=gorm DB.name=gorm port=9920 sslmode=disable")
	// db, err = gorm.Open("mysql", "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True")
	// db, err = gorm.Open("mssql", "sqlserver://gorm:LoremIpsum86@localhost:9930?database=gorm")
	if err != nil {
		panic(err)
	}
	db.LogMode(true)
}

func main() {
	if /* failure condition */ {
		fmt.Println("failed")
	} else {
		fmt.Println("success")
	}
}
```
