# GORM

Easy-to-use ORM lib for Golang.

[![go report card](https://goreportcard.com/badge/github.com/go-gorm/gorm "go report card")](https://goreportcard.com/report/github.com/go-gorm/gorm)
[![test status](https://github.com/go-gorm/gorm/workflows/tests/badge.svg?branch=master "test status")](https://github.com/go-gorm/gorm/actions)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/gorm.io/gorm?tab=doc)

## Highlights

* Full-featured ORM
* Relationship types (Has one, Has many, Belong to, many to many, polymorphism, single-table inheritance)
* Hooks (Before/After, Create/Save/Update/Delete/Find)
* Eager loading by `Preload`, `Joins`
* Transaction, nested transactions, savepoint, RollbackTo
* Context, prepared statement mode, DryRun mode
* Batch insert, FindInBatches, Find to map
* SQL builder, upsert, locking, optimizer/index/comment hints, NamedArg, Search/Update/Create with SQL expr
* Composite primary key
* Migration
* Logger
* Extendable, flexible plugin API: Database resolver (Multiple databases, read/write splitting) / Prometheus…


## Getting started

* GORM guide [https://gorm.io](https://gorm.io)
* Gen guide [https://gorm.io/gen/index.html](https://gorm.io/gen/index.html)

## Contribute

[You can help to deliver a better GORM, check out things you can do](https://gorm.io/contribute.html)

Thank you for contributing to the GORM framework!

[![Contributors](https://contrib.rocks/image?repo=go-gorm/gorm)](https://github.com/go-gorm/gorm/graphs/contributors)

## License

© Jinzhu and GORM's committer, 2013 - 2023

Released under the [MIT License](https://github.com/go-gorm/gorm/blob/master/License)
