# GORM

Yet Another ORM library for Go, aims for developer friendly

## USAGE

```go
db, _ = Open("postgres", "user=gorm dbname=gorm sslmode=disable")

type User struct {
	Id        int64
	Age       int64
	Birthday  time.Time
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Create
user = User{Name: "jinzhu", Age: 18, Birthday: time.Now())
db.Save(&user)

// Update
user.Name = "jinzhu 2"
db.Save(&user)

// Delete
db.Delete(&user)

// Get First matched record
db.Where("name = ?", "jinzhu").First(&user)

// Get All matched records
db.Where("name = ?", "jinzhu").Find(&users)

// Advanced Where Usage
db.Where("name <> ?", "jinzhu").Find(&users)
db.Where("name = ? and age >= ?", "3", "22").Find(&users)
db.Where("name in (?)", []string["jinzhu", "jinzhu 2"]).Find(&users)
db.Where("birthday < ?", time.Now()).Find(&users)

// Select
db.Select("name").Find(&users)

// Order
db.Order("age desc, name").Find(&users)

// Limit
db.Limit(3).Find(&users)

// Offset
db.Offset(3).Find(&users)

// Or
db.Where("name = ?", "Jinzhu").Or("name = ?", "Jinzhu 2").Find(&users)

// Count
db.Where("name = ?", "1").Or("name = ?", "3").Find(&users).Count(&count)
db.Model(&User{}).Where("name = ?", "1").Count(&count)

// CreatedAt (auto insert current time on create)
If your struct has field CreatedAt, it will be filled with the current time when insert into database

// UpdatedAt (auto update the time on save)
If your struct has field UpdatedAt, it will be filled with the current time when update it

// Callbacks
Below callbacks defined now:
BeforeCreate, BeforeUpdate, BeforeSave, AfterCreate, AfterUpdate, AfterSave
BeforeDelete, AfterDelete

Callbacks is a function defined to a model, if the function return error, will prevent the database operations.

// Pluck (get all users's age as map)
var ages []int64
db.Model(&User{}).Pluck("age", &ages)

// Query Chains
db.Where("name <> ?", "jinzhu").Where("age >= ? and role <> ?", 20, "admin").Find(&users)

// Create Table with struct
db.CreateTable(&User{})

// Run Raw SQL
db.Exec("drop table users;")
```

## TODO
* Update, Updates
* Soft Delete
* Even more complex where query (with map or struct)
* FindOrInitialize / FindOrCreate
* SQL Log
* Auto Migration
* Index, Unique, Valiations

# Author

**Jinzhu**

* <http://github.com/jinzhu>
* <wosmvp@gmail.com>
* <http://twitter.com/zhangjinzhu>
