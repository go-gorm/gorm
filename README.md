# GORM

Yet Another ORM library for Go, aims for developer friendly

## Overview

* CURD
* Chainable API
* Before/After Create/Save/Update/Delete Callbacks
* Order/Select/Limit/Offset Support
* Update, Updates Like Rails's update_attribute, update_attributes
* FirstOrInit, FirstOrCreate Like Rails's first_or_initialize, first_or_create
* Dynamically set table name when search, update, delete...
* Automatically CreatedAt, UpdatedAt
* Soft Delete
* Create table from struct
* Prevent SQL Injection
* Goroutines friendly
* Database Pool

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

// Set database pool
db.SetPool(10)

// Create
user = User{Name: "jinzhu", Age: 18, Birthday: time.Now()}
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
db.Where(20).First(&user)
//// users -> select * from users where id = 20;
db.Where([]int64{20, 21, 22}).Find(&user)
//// users -> select * from users where id in (20, 21, 22);
db.Where("name = ? and age >= ?", "jinzhu", "22").Find(&users)
//// users -> select * from users name = 'jinzhu' and age >= 22;
db.Where("name in (?)", []string["jinzhu", "jinzhu 2"]).Find(&users)
//// users -> select * from users name in ('jinzhu', 'jinzhu 2');
db.Where("name LIKE ?", "%jin%").Find(&users)
//// users -> select * from users name LIKE "%jinzhu%";
db.Where(&User{Name: "jinzhu", Age: 20}).First(&user)
//// user -> select * from users name = "jinzhu" and age = 20 limit 1;
db.Where(map[string]interface{}{"name": "jinzhu", "age": 20}).First(&user)
//// user -> select * from users name = "jinzhu" and age = 20 limit 1;
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
db.Find(&users, &User{Age: 20})
//// users -> select * from users where age = 20;
db.Find(&users, map[string]interface{}{"age": 20})
//// users -> select * from users where age = 20;

// FirstOrInit
db.FirstOrInit(&user, User{Name: "noexisting_user"})
//// user -> User{Name: "noexisting_user"}
db.Where(User{Name: "Jinzhu"}).FirstOrInit(&user)
//// user -> User{Id: 111, Name: "Jinzhu"}
db.FirstOrInit(&user, map[string]interface{}{"name": "jinzhu", "age": 20})
//// user -> User{Id: 111, Name: "Jinzhu", Age: 20}

// FirstOrInit With Attrs
db.Where(User{Name: "noexisting_user"}).Attrs(User{Age: 20}).FirstOrInit(&user)
//// user -> select * from users where name = 'noexisting_user';
//// If no record found, will assign the attrs to user, so user become:
////   User{Name: "noexisting_user", Age: 20}
db.Where(User{Name: "noexisting_user"}).Attrs("age", 20).FirstOrInit(&user)
// Same as above
db.Where(User{Name: "Jinzhu"}).Attrs(User{Age: 20}).FirstOrInit(&user)
//// user -> select * from users where name = 'jinzhu';
//// If found the user, will ingore the attrs:
////   User{Id: 111, Name: "Jinzhu", Age: 18}

// FirstOrInit With Assign
db.Where(User{Name: "noexisting_user"}).Assign(User{Age: 20}).FirstOrInit(&user)
//// user -> select * from users where name = 'noexisting_user';
//// If no record found, will assign the value to user, so user become:
////   User{Name: "noexisting_user", Age: 20} (same as FirstOrInit With Attrs)
db.Where(User{Name: "noexisting_user"}).Assign("age", 20).FirstOrInit(&user)
// Same as above
//// user -> User{Name: "noexisting_user", Age: 20}
db.Where(User{Name: "Jinzhu"}).Assign(User{Age: 20}).FirstOrInit(&user)
//// user -> select * from users where name = 'jinzhu';
//// If found the user, will assign the value to user, so user become: (different with FirstOrInit With Attrs)
////   User{Id: 111, Name: "Jinzhu", Age: 20}

// FirstOrCreate
db.FirstOrCreate(&user, User{Name: "noexisting_user"})
//// user -> User{Id: 112, Name: "noexisting_user"}
db.Where(User{Name: "Jinzhu"}).FirstOrCreate(&user)
//// user -> User{Id: 111, Name: "Jinzhu"}
db.FirstOrCreate(&user, map[string]interface{}{"name": "jinzhu", "age": 20})
//// user -> User{Id: 111, Name: "Jinzhu", Age: 20}

// FirstOrCreate With Attrs
db.Where(User{Name: "noexisting_user"}).Attrs(User{Age: 20}).FirstOrCreate(&user)
//// user -> select * from users where name = 'noexisting_user';
//// If not record found, will assing the attrs to the user first, then create it
//// Same as db.Where(User{Name: "noexisting_user"}).FirstOrCreate(&user).Update("age": 20), but one less sql
db.Where(User{Name: "noexisting_user"}).Attrs("age", 20).FirstOrCreate(&user)
// Save as above
//// user -> User{Id: 112, Name: "noexisting_user", Age: 20}
db.Where(User{Name: "Jinzhu"}).Attrs(User{Age: 20}).FirstOrCreate(&user)
//// user -> select * from users where name = 'jinzhu';
//// If found any record, will ignore the attrs
//// user -> User{Id: 111, Name: "Jinzhu", Age: 18}

// FirstOrCreate With Assign
db.Where(User{Name: "noexisting_user"}).Assign(User{Age: 20}).FirstOrCreate(&user)
//// user -> select * from users where name = 'noexisting_user';
//// If not record found, will assing the value to the user first, then create it
//// user -> User{Id: 112, Name: "noexisting_user", Age: 20} (Same as FirstOrCreate With Attrs)
db.Where(User{Name: "Jinzhu"}).Assign(User{Age: 20}).FirstOrCreate(&user)
//// user -> select * from users where name = 'jinzhu';
//// If any record found, will assing the value to the user and update it
//// UPDATE users SET age=20 WHERE id = 111;
//// user -> User{Id: 111, Name: "Jinzhu", Age: 20}


//// user -> User{Id: 111, Name: "Jinzhu", Age: 18}
//// You must noticed that the Attrs is similar to FirstOrInit with Attrs, yes?

// Select
db.Select("name").Find(&users)
//// users -> select name from users;

// Order
db.Order("age desc, name").Find(&users)
//// users -> select * from users order by age desc, name;
db.Order("age desc").Order("name").Find(&users)
//// users -> select * from users order by age desc, name;
// ReOrder
db.Order("age desc").Find(&users1).Order("age", true).Find(&users2)
//// users1 -> select * from users order by age desc;
//// users2 -> select * from users order by age;

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

// Specify Table Name
db.Table("deleted_users").CreateTable(&User{})
db.Table("users").Pluck("age", &ages)
//// ages -> select age from users;
var deleted_users []User
db.Table("deleted_users").Find(&deleted_users)
//// deleted_users -> select * from deleted_users;
db.Table("deleted_users").Find(&deleted_user)
//// deleted_user -> select * from deleted_users limit 1;

// Update
db.Table("users").Where(10).Update("name", "hello")
//// update users set name='hello' where id = 10;
db.Table("users").Update("name", "hello")
//// update users set name='hello';

// Updates
db.Table("users").Where(10).Updates(map[string]interface{}{"name": "hello", "age": 18})
//// update users set name='hello', age=18 where id = 10;
db.Table("users").Updates(map[string]interface{}{"name": "hello", "age": 18})
//// update users set name='hello', age=18;
db.Find(&users).Updates(User{Name: "hello", Age: 18})
//// update users set name='hello', age=18;
db.First(&user, 20).Updates(User{Name: "hello", Age: 18})
//// update users set name='hello', age=18 where id = 20;
//// object user's value would be reflected by the Updates also,
//// so you don't need to refetch the user from database

// Soft Delete
// For those struct have DeletedAt field, they will get soft delete ability automatically!
type Order struct {
  Id        int64
  Amount    int64
  CreatedAt time.Time
  UpdatedAt time.Time
  DeletedAt time.Time
}
order := order{Id:10}
db.Delete(&order)
//// UPDATE orders SET deleted_at="2013-10-29 10:23" WHERE id = 10;
db.Where("amount = ?", 0).Delete(&Order{})
//// UPDATE orders SET deleted_at="2013-10-29 10:23" WHERE amount = 0;
db.Where("amount = 100").Find(&order)
//// order -> select * from orders where amount = 100 and (deleted_at is null and deleted_at <= '0001-01-02');
// And you are possible to query soft deleted orders with Unscoped method
db.Unscoped().Where("amount = 100").Find(&order)
//// order -> select * from orders where amount = 100;
// Of course, you could permanently delete a record with Unscoped
db.Unscoped().Delete(&order)
// DELETE from orders where id=10;

// Run Raw SQL
db.Exec("drop table users;")

// Error Handling
query := db.Where("name = ?", "jinzhu").First(&user)
query := db.First(&user).Limit(10).Find(&users)
//// query.Error -> the last error happened
//// query.Errors -> all errors happened
//// If an error happened, gorm will stop do insert, update, delete operations
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

db.Where("mail_type = ?", "TEXT").Find(&users1).Table("deleted_users").First(&user2)
//// users1 -> select * from users where mail_type = 'TEXT';
//// users2 -> select * from deleted_users where mail_type = 'TEXT';

db.Where("email = ?", "x@example.org"').Attrs(User{FromIp: "111.111.111.111"}).FirstOrCreate(&user)
//// user -> select * from users where email = 'x@example.org'
//// (if no record found) -> INSERT INTO "users" (email,from_ip) VALUES ("x@example.org", "111.111.111.111")

// Open your mind, add more cool examples
```

## TODO
* SubStruct
* Index, Unique, Valiations
* Auto Migration
* SQL Log
* SQL Query with goroutines
* Only tested with postgres, confirm works with other database adaptors

# Author

**Jinzhu**

* <http://github.com/jinzhu>
* <wosmvp@gmail.com>
* <http://twitter.com/zhangjinzhu>
