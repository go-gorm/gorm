# GORM

The fantastic ORM library for Golang, aims to be developer friendly.

[![Join the chat at https://gitter.im/jinzhu/gorm](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/jinzhu/gorm?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![wercker status](https://app.wercker.com/status/0cb7bb1039e21b74f8274941428e0921/s/master "wercker status")](https://app.wercker.com/project/bykey/0cb7bb1039e21b74f8274941428e0921)
[![GoDoc](https://godoc.org/github.com/jinzhu/gorm?status.svg)](https://godoc.org/github.com/jinzhu/gorm)

## Overview

* Full-Featured ORM (almost)
* Associations (Has One, Has Many, Belongs To, Many To Many, Polymorphism)
* Callbacks (Before/After Create/Save/Update/Delete/Find)
* Preloading (eager loading)
* Transactions
* Composite Primary Key
* SQL Builder
* Auto Migrations
* Logger
* Extendable, write Plugins based on GORM callbacks
* Every feature comes with tests
* Developer Friendly

## Install

```sh
go get -u github.com/jinzhu/gorm
```

## Upgrading To V1.0

* [CHANGELOG](changelog.html)

## Quick Start

```go
package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Product struct {
  gorm.Model
  Code string
  Price uint
}

func main() {
  db, err := gorm.Open("sqlite", "test.db")
  if err != nil {
    panic("faield to connect database")
  }

  // Create
  db.Create(&Product{Code: "L1212", Price: 1000})

  // Read
  var product Product
  db.First(&product, 1) // find product with id 1
  db.First(&product, "code = ?", "L1212") // find product with code l1212

  // Update - update product's price to 2000
  db.Model(&product).Update("Price", 2000)

  // Delete - delete product
  db.Delete(&product)
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
