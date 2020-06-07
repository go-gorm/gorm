Your issue may already be reported! Please search on the [issue track](https://github.com/go-gorm/gorm/issues) before creating one.

### What version of Go are you using (`go version`)?


### Which database and its version are you using?


### Please provide a complete runnable program to reproduce your issue. **IMPORTANT**

Need to runnable with [GORM's docker compose config](https://github.com/go-gorm/gorm/blob/master/tests/docker-compose.yml) or please provides your config.

```go
package main

import (
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
//  "gorm.io/driver/mysql"
//  "gorm.io/driver/postgres"
//  "gorm.io/driver/sqlserver"
)

func main() {
  db, err := gorm.Open(sqlite.Open(filepath.Join(os.TempDir(), "gorm.db")), &gorm.Config{})
  // db, err := gorm.Open(postgres.Open("user=gorm password=gorm DB.name=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"), &gorm.Config{})
  // db, err := gorm.Open(mysql.Open("gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local"), &gorm.Config{})
  // db, err := gorm.Open(sqlserver.Open("sqlserver://gorm:LoremIpsum86@localhost:9930?database=gorm"), &gorm.Config{})

  /* your code */

	if /* failure condition */ {
		fmt.Println("failed")
	} else {
		fmt.Println("success")
	}
}
```
