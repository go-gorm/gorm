# GORM

Yet Another ORM library for Go, aims for developer friendly

## Overview

* CURD
* Chainable API
* Embedded structs support
* Before/After Create/Save/Update/Delete Callbacks
* Update, Updates Like Rails's update_attribute, update_attributes
* FirstOrInit, FirstOrCreate Like Rails's first_or_initialize, first_or_create
* Order/Select/Limit/Offset Support
* Automatically CreatedAt, UpdatedAt
* Soft Delete
* Create/Drop table from struct
* Dynamically set table name when search, create, update, delete...
* Prevent SQL Injection
* Goroutines friendly
* Database Pool
* Convention Over Configuration (CoC)

## Basic Usage

## Opening a Database

```go
db, err = Open("postgres", "user=gorm dbname=gorm sslmode=disable")

// Gorm is goroutines friendly, so you can create a global variable to keep the connection and use it everywhere

var DB gorm.DB

func init() {
    DB, err = gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
    if err != nil {
        panic(fmt.Sprintf("Got error when connect database, the error is '%v'", err))
    }
}

// Set the maximum idle database connections
db.SetPool(100)
```

## Conventions

```go
type User struct {              // TableName: `users`, gorm will pluralize struct name as table name
	Id                int64     // Id: Database Primary key
	Birthday          time.Time // Time
	Age               int64
	Name              string
	CreatedAt         time.Time // CreatedAt: Time of record is created, will be insert automatically
	UpdatedAt         time.Time // UpdatedAt: Time of record is updated, will be updated automatically
	DeletedAt         time.Time // DeletedAt: Time of record is deleted, refer Soft Delete for more
	Email             []Email   // Embedded structs
	BillingAddress    Address   // Embedded struct
	BillingAddressId  int64     // Embedded struct's foreign key
	ShippingAddress   Address   // Embedded struct
	ShippingAddressId int64     // Embedded struct's foreign key
}

type Email struct {    // TableName: `emails`
	Id         int64
	UserId     int64   // Foreign key for above embedded structs
	Email      string
	Subscribed bool
}

type Address struct {  // TableName: `addresses`
	Id       int64
	Address1 string
	Address2 string
	Post     string
}
```

## Struct & Database Mapping

```go
// Create table from struct
db.CreateTable(User{})

// Drop table
db.DropTable(User{})
```

## Create

```go
user := User{Name: "jinzhu", Age: 18, Birthday: time.Now()}
db.Save(&user)
```

## Update

```go
user.Name = "jinzhu 2"
db.Save(&user)
```

## Delete

```go
db.Delete(&user)
```

## Query

```go
// Get the first record
db.First(&user)
//// SELECT * FROM USERS LIMIT 1;

// Get All records
db.Find(&users)
//// SELECT * FROM USERS;

// Using a Primary Key
db.First(&user, 10)
//// SELECT * FROM USERS WHERE id = 10;
```

### Where (SQL like condition)

```go
// Get the first matched record
db.Where("name = ?", "jinzhu").First(&user)
//// SELECT * FROM users WHERE name = 'jinzhu' limit 1;

// Get all matched records
db.Where("name = ?", "jinzhu").Find(&users)
//// SELECT * FROM users WHERE name = 'jinzhu';

db.Where("name <> ?", "jinzhu").Find(&users)
//// SELECT * FROM users WHERE name <> 'jinzhu';

// IN
db.Where("name in (?)", []string["jinzhu", "jinzhu 2"]).Find(&users)
//// SELECT * FROM users WHERE name IN ('jinzhu', 'jinzhu 2');

// IN For Primary Key
db.Where([]int64{20, 21, 22}).Find(&users)
//// SELECT * FROM users WHERE id IN (20, 21, 22);

// LIKE
db.Where("name LIKE ?", "%jin%").Find(&users)
//// SELECT * FROM users WHERE name LIKE "%jin%";

// Multiple Conditions
db.Where("name = ? and age >= ?", "jinzhu", "22").Find(&users)
//// SELECT * FROM users WHERE name = 'jinzhu' AND age >= 22;
```

### Where (Struct & Map)

```go
// Search with struct
db.Where(&User{Name: "jinzhu", Age: 20}).First(&user)
//// SELECT * FROM users WHERE name = "jinzhu" AND age = 20 LIMIT 1;

// Search with map
db.Where(map[string]interface{}{"name": "jinzhu", "age": 20}).Find(&users)
//// SELECT * FROM users WHERE name = "jinzhu" AND age = 20;
```

### Not

```go
// Not Equal
db.Not("name", "jinzhu").First(&user)
//// SELECT * FROM users WHERE name <> "jinzhu" LIMIT 1;

// Not In
db.Not("name", []string{"jinzhu", "jinzhu 2"}).Find(&users)
//// SELECT * FROM users WHERE name NOT IN ("jinzhu", "jinzhu 2");

// Not In for Primary Key
db.Not([]int64{1,2,3}).First(&user)
//// SELECT * FROM users WHERE id NOT IN (1,2,3);

db.Not([]int64{}).First(&user)
//// SELECT * FROM users;

// Normal SQL
db.Not("name = ?", "jinzhu").First(&user)
//// SELECT * FROM users WHERE NOT(name = "jinzhu");

// Not With Struct
db.Not(User{Name: "jinzhu"}).First(&user)
//// SELECT * FROM users WHERE name <> "jinzhu";
```

### Inline Search Condition

```go
// Find with primary key
db.First(&user, 23)
//// SELECT * FROM users WHERE id = 23 LIMIT 1;

// Normal SQL
db.Find(&user, "name = ?", "jinzhu")
//// SELECT * FROM users WHERE name = "jinzhu";

// Multiple Conditions
db.Find(&users, "name <> ? and age > ?", "jinzhu", 20)
//// SELECT * FROM users WHERE name <> "jinzhu" AND age > 20;

// Inline Search With Struct
db.Find(&users, User{Age: 20})
//// SELECT * FROM users WHERE age = 20;

// Inline Search With Map
db.Find(&users, map[string]interface{}{"age": 20})
//// SELECT * FROM users WHERE age = 20;
```

## FirstOrInit

Try to load the first record, if fails, initialize struct with search conditions.
(only support map or struct conditions, SQL like conditions are not supported)

```go
db.FirstOrInit(&user, User{Name: "non_existing"})
//// User{Name: "non_existing"}

db.Where(User{Name: "Jinzhu"}).FirstOrInit(&user)
//// User{Id: 111, Name: "Jinzhu", Age: 20}

db.FirstOrInit(&user, map[string]interface{}{"name": "jinzhu"})
//// User{Id: 111, Name: "Jinzhu", Age: 20}
```

### FirstOrInit With Attrs

Attr's arguments would be used to initialize struct if not record found, but won't be used for search

```go
db.Where(User{Name: "non_existing"}).Attrs(User{Age: 20}).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = 'non_existing';
//// User{Name: "non_existing", Age: 20}

// Above code could be simplified if have only one attribute
db.Where(User{Name: "noexisting_user"}).Attrs("age", 20).FirstOrInit(&user)

// If record found, Attrs would be ignored
db.Where(User{Name: "Jinzhu"}).Attrs(User{Age: 30}).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = jinzhu';
////   User{Id: 111, Name: "Jinzhu", Age: 20}

### FirstOrInit With Assign

Assign's arguments would be used to set the struct even record found, but won't be used for search

```go
db.Where(User{Name: "non_existing"}).Assign(User{Age: 20}).FirstOrInit(&user)
//// User{Name: "non_existing", Age: 20}

db.Where(User{Name: "Jinzhu"}).Assign(User{Age: 30}).FirstOrInit(&user)
//// User{Id: 111, Name: "Jinzhu", Age: 30}
```

## FirstOrCreate

Try to load the first record, if fails, initialize struct with search conditions and save it

```go
db.FirstOrCreate(&user, User{Name: "non_existing"})
//// User{Id: 112, Name: "non_existing"}

db.Where(User{Name: "Jinzhu"}).FirstOrCreate(&user)
//// User{Id: 111, Name: "Jinzhu"}

db.FirstOrCreate(&user, map[string]interface{}{"name": "jinzhu", "age": 30})
//// user -> User{Id: 111, Name: "Jinzhu", Age: 20}
```

### FirstOrCreate With Attrs

Attr's arguments would be used to initialize struct if not record found, but won't be used for search

```go
// FirstOrCreate With Attrs
db.Where(User{Name: "non_existing"}).Attrs(User{Age: 20}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE name = 'non_existing';
//// User{Id: 112, Name: "non_existing", Age: 20}

db.Where(User{Name: "Jinzhu"}).Attrs(User{Age: 30}).FirstOrCreate(&user)
//// User{Id: 111, Name: "Jinzhu", Age: 20}
```

### FirstOrCreate With Assign

Assign's arguments would be used to initialize the struct if not record found,
If any record found, will assign those values to the record, and save it back to database.

```go
db.Where(User{Name: "non_existing"}).Assign(User{Age: 20}).FirstOrCreate(&user)
//// user -> User{Id: 112, Name: "non_existing", Age: 20}

db.Where(User{Name: "Jinzhu"}).Assign(User{Age: 30}).FirstOrCreate(&user)
//// UPDATE users SET age=30 WHERE id = 111;
//// User{Id: 111, Name: "Jinzhu", Age: 30}
```

### SELECT

```go
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

// Drop Table
db.DropTable(&User{})

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
