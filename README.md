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
//// users -> select * from users name <> 'jinzhu';
db.Where("name = ? and age >= ?", "jinzhu", "22").Find(&users)
//// users -> select * from users name = 'jinzhu' and age >= 22;
db.Where("name in (?)", []string["jinzhu", "jinzhu 2"]).Find(&users)
//// users -> select * from users name in ('jinzhu', 'jinzhu 2');
db.Where("name LIKE ?", "%jin%").Find(&users)
//// users -> select * from users name LIKE "%jinzhu%";
db.Where("birthday < ?", time.Now()).Find(&users)

// Inline search condition
db.First(&user, 23)
//// user -> select * from users where id = 23 limit 1;
db.First(&user, "name = ?", "jinzhu")
//// user -> select * from users where name = "jinzhu" limit 1;
db.Find(&users, "name = ?", "jinzhu")
//// users -> select * from users where name = "jinzhu";
db.Find(&users, "name <> ? and age > ?", "jinzhu", 20)
//// users -> select * from users where name <> "jinzhu" and age > 20;

// Select
db.Select("name").Find(&users)
//// users -> select name from users;

// Order
db.Order("age desc, name").Find(&users)
//// users -> select * from users order by age desc, name;
db.Order("age desc").Order("name").Find(&users)
//// users -> select * from users order by age desc, name;

// Limit
db.Limit(3).Find(&users)
//// users -> select * from users limit 3;
db.Limit(10).Find(&users1).Limit(20).Find(&users2).Limit(-1).Find(&users3)
//// users1 -> select * from users limit 10;
//// users2 -> select * from users limit 20;
//// users3 -> select * from users;


// Offset
//// select * from users offset 3;
db.Offset(3).Find(&users)
db.Offset(10).Find(&users1).Offset(20).Find(&users2).Offset(-1).Find(&users3)
//// user1 -> select * from users offset 10;
//// user2 -> select * from users offset 20;
//// user3 -> select * from users;

// Or
db.Where("role = ?", "admin").Or("role = ?", "super_admin").Find(&users)
//// users -> select * from users where role = 'admin' or role = 'super_admin';

// Count
db.Where("name = ?", "jinzhu").Or("name = ?", "jinzhu 2").Find(&users).Count(&count)
//// users -> select * from users where name = 'jinzhu' or name = 'jinzhu 2';
//// count -> select count(*) from users where name = 'jinzhu' or name = 'jinzhu 2';
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
//// ages -> select age from users;
var names []string
db.Model(&User{}).Pluck("name", &names)
//// names -> select name from users;

// Query Chains
db.Where("name <> ?", "jinzhu").Where("age >= ? and role <> ?", 20, "admin").Find(&users)
//// users -> select * from users where name <> 'jinzhu' andd age >= 20 and role <> 'admin';

// Create Table with struct
db.CreateTable(&User{})

// Run Raw SQL
db.Exec("drop table users;")
```

## Advanced Usage With Query Chain

```go
// Already excited about the basic usage? Let's see some magic!

db.First(&first_article).Count(&total_count).Limit(10).Find(&first_page_articles).Offset(10).Find(&second_page_articles)
//// first_article -> select * from articles limit 1
//// total_count -> select count(*) from articles
//// first_page_articles -> select * from articles limit 10
//// second_page_articles -> select * from articles limit 10 offset 10

db.Where("created_at > ?", "2013/10/10").Find(&cancelled_orders, "state = ?", "cancelled").Find(&shipped_orders, "state = ?", "shipped")
//// cancelled_orders -> select * from orders where created_at > '2013/10/10' and state = 'cancelled'
//// shipped_orders -> select * from orders where created_at > '2013/10/10' and state = 'shipped'

db.Model(&Order{}).Where("amount > ?", 10000).Pluck("user_id", &paid_user_ids)
//// paid_user_ids -> select user_id from orders where amount > 10000
db.Where("user_id = ?", paid_user_ids).Find(&:paid_users)
//// paid_users -> select * from users where user_id in (10, 20, 99)

db.Where("product_name = ?", "fancy_product").Find(&orders).Find(&shopping_cart)
//// orders -> select * from orders where product_name = 'fancy_product'
//// shopping_cart -> select * from carts where product_name = 'fancy_product'
// Do you noticed the search table is different for above query, yay

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
* SQL Query with goroutines
* Only tested with postgres, confirm works with other database adaptors

# Author

**Jinzhu**

* <http://github.com/jinzhu>
* <wosmvp@gmail.com>
* <http://twitter.com/zhangjinzhu>
