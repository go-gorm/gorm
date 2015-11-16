# How to Contribute

## Bug Report

- Do a search on GitHub under Issues in case it has already been reported
- Submit __executable script__ or failing test pull request that could demonstrates the issue is *MUST HAVE*

## Feature Request

- Feature request with pull request is welcome
- Or it won't be implemented until I (other developers) find it is helpful for my (their) daily work

## Pull Request

- Prefer single commit pull request, that make the git history can be a bit easier to follow.
- New features need to be covered with tests to make sure your code works as expected, and won't be broken by others in future

## Contributing to Documentation

- You are welcome ;)
- You can help improve the README by making them more coherent, consistent or readable, and add more godoc documents to make people easier to follow.
- Blogs & Usage Guides & PPT also welcome, please add them to https://github.com/jinzhu/gorm/wiki/Guides

### Executable script template

```go
package main

import (
    _ "github.com/mattn/go-sqlite3"
    _ "github.com/go-sql-driver/mysql"
    _ "github.com/lib/pq"
    "github.com/jinzhu/gorm"
)

var db gorm.DB

func init() {
    var err error
    db, err = gorm.Open("sqlite3", "test.db")
    // db, err := gorm.Open("postgres", "user=username dbname=password sslmode=disable")
    // db, err := gorm.Open("mysql", "user:password@/dbname?charset=utf8&parseTime=True")
    if err != nil {
        panic(err)
    }
    db.LogMode(true)
}

func main() {
    // Your code
}
```
