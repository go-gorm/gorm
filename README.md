# GORM

Yet Another ORM library for Go, aims for developer friendly

## Basic Usage

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

// Inline search condition
db.First(&user, 23) // select * from users where id = 23 limit 1;
db.First(&user, "name = ?", "jinzhu") // select * from users where name = "jinzhu" limit 1;
db.Find(&users, "name = ?", "jinzhu") // select * from users where name = "jinzhu";
db.Find(&users, "name <> ? and age > ?", "jinzhu", 20) // select * from users where name <> "jinzhu" and age > 20;

// Select
db.Select("name").Find(&users)

// Order
db.Order("age desc, name").Find(&users)
db.Order("age desc").Order("name").Find(&users)

// Limit
db.Limit(3).Find(&users)
db.Limit(10).Find(&ten_users).Limit(20).Find(&twenty_users).Limit(-1).Find(&all_users)

// Offset
db.Offset(3).Find(&users)
db.Offset(10).Find(&users).Offset(20).Find(&users).Offset(-1).Find(&users)

// Or
db.Where("role = ?", "admin").Or("role = ?", "super_admin").Find(&users)

// Count
db.Where("name = ?", "jinzhu").Or("name = ?", "jinzhu 2").Find(&users).Count(&count)
db.Model(&User{}).Where("name = ?", "jinzhu").Count(&count)

// CreatedAt (auto insert current time on create)
If your struct has field CreatedAt,
it will be filled with the current time when insert into database

// UpdatedAt (auto update the time on save)
If your struct has field UpdatedAt,
it will be filled with the current time when update it

// Callbacks
Below callbacks are defined now:

`BeforeCreate`, `BeforeUpdate`, `BeforeSave`, `AfterCreate`, `AfterUpdate`, `AfterSave`
`BeforeDelete`, `AfterDelete`

Callbacks is a function defined to a model, if the function return error, will prevent the database operations.

  func (u *User) BeforeUpdate() (err error) {
      if u.readonly() {
          err = errors.New("Read Only User")
      }
      return
  }

// Pluck (get users's age as map)
var ages []int64
db.Find(&users).Pluck("age", &ages)
var names []string
db.Model(&User{}).Pluck("name", &names)

// Query Chains
db.Where("name <> ?", "jinzhu").Where("age >= ? and role <> ?", 20, "admin").Find(&users)

// Create Table with struct
db.CreateTable(&User{})

// Run Raw SQL
db.Exec("drop table users;")
```

## Cool Examples

```go
// Already excited about the basic usage? Let's see some magic!

db.First(&first_article).Count(&total_count).Limit(10).Find(&first_page_articles).Offset(10).Find(&second_page_articles)
// first_article return the latest article
// total_count return the total numbers of articles
// first_page_articles return the latest 10 articles
// second_page_articles return the latest 10 to 20 articles

db.Where("created_at > ?", "2013/10/10").Find(&cancelled_orders, "state = ?", "cancelled").Find(&shipped_orders, "state = ?", "shipped")
// cancelled_orders return all cancelled orders since 2013/10/10
// shipped_orders return all shipped orders since 2013/10/10

db.Model(&Order{}).Where("amount > ?", 10000).Pluck("user_id", &paid_user_ids)
db.Where("user_id = ?", paid_user_ids).Find(&:paid_users)

db.Where("product_name = ?", "fancy_product").Find(&orders).Find(&shopping_cart)

// Open your mind, add more cool examples
```

## TODO
* Update, Updates like rails's update_attribute, update_attributes
* Soft Delete
* Query with map or struct
* FindOrInitialize / FindOrCreate
* SQL Log
* Auto Migration
* Index, Unique, Valiations

# Author

**Jinzhu**

* <http://github.com/jinzhu>
* <wosmvp@gmail.com>
* <http://twitter.com/zhangjinzhu>
