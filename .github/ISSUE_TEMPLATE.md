Before posting a bug report about a problem, please try to verify that it is a bug and that it has not been reported already, please apply corresponding GitHub labels to the issue, for feature requests, please apply `type:feature`.

DON'T post usage related questions, ask in https://gitter.im/jinzhu/gorm or http://stackoverflow.com/questions/tagged/go-gorm,

Please answer these questions before submitting your issue. Thanks!



### What version of Go are you using (`go version`)?


### Which database and its version are you using?


### What did you do?

Please provide a complete runnable program to reproduce your issue.

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
	// Please use below username, password as your database's account for the script.
	// db, err = gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
	// db, err = gorm.Open("mysql", "gorm:gorm@/dbname?charset=utf8&parseTime=True")
	// db, err = gorm.Open("mssql", "sqlserver://gorm:LoremIpsum86@localhost:1433?database=gorm")
	if err != nil {
		panic(err)
	}
	db.LogMode(true)
}

func main() {
	// your code here

	if /* failure condition */ {
		fmt.Println("failed")
	} else {
		fmt.Println("success")
	}
}
```
