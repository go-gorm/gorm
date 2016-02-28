# GORM

The fantastic ORM library for Golang, aims to be developer friendly.

[![wercker status](https://app.wercker.com/status/0cb7bb1039e21b74f8274941428e0921/s/master "wercker status")](https://app.wercker.com/project/bykey/0cb7bb1039e21b74f8274941428e0921)
[![GoDoc](https://godoc.org/github.com/jinzhu/gorm?status.svg)](https://godoc.org/github.com/jinzhu/gorm)
[![Join the chat at https://gitter.im/jinzhu/gorm](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/jinzhu/gorm?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Overview

* Full-Featured ORM (almost)
* Chainable API
* Auto Migrations
* Relations (Has One, Has Many, Belongs To, Many To Many, [Polymorphism](#polymorphism))
* Callbacks (Before/After Create/Save/Update/Delete/Find)
* Preloading (eager loading)
* Transactions
* Embed Anonymous Struct
* Soft Deletes
* Customizable Logger
* Iteration Support via [Rows](#row--rows)
* Every feature comes with tests
* Developer Friendly

## Install

```
go get -u github.com/jinzhu/gorm
```

## Basic Usage

```go
type Product struct {
  gorm.Model
  Code string
  Price uint
}

var db *gorm.DB

func init() {
  var err error
  db, err = gorm.Open("sqlite", "test.db")
}

func main() {
  db.Create(&Product{Code: "L1212", Price: 1000})

  var product Product
  db.First(&product, 1) // find product with id 1
  db.First(&product, "code = ?", "L1212") // find product with code l1212

  db.Model(&product).Update("Price", 2000) // update product's price to 2000

  db.Delete(&product) // delete product
}
```

# Author

**jinzhu**

* <http://github.com/jinzhu>
* <wosmvp@gmail.com>
* <http://twitter.com/zhangjinzhu>

# Contributors

https://github.com/jinzhu/gorm/graphs/contributors

## License

Released under the [MIT License](https://github.com/jinzhu/gorm/blob/master/License).
