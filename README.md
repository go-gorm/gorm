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
* Convention Over Configuration

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
	BillingAddressId  int64     // Embedded struct BillingAddress's foreign key
	ShippingAddress   Address   // Embedded struct
	ShippingAddressId int64     // Embedded struct ShippingAddress's foreign key
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

## Query

```go
// Get the first record
db.First(&user)
//// SELECT * FROM users LIMIT 1;
// Search table `users` are guessed from the out struct's name.
// You are possible to specify the table name with Model() if no out struct for some methods like Pluck()
// Or set table name with Table(), if so, it will ignore the out struct's type even have it. more details later.

// Get All records
db.Find(&users)
//// SELECT * FROM users;

// Using a Primary Key
db.First(&user, 10)
//// SELECT * FROM users WHERE id = 10;
```

### Query With Where (SQL like condition)

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

// LIKE
db.Where("name LIKE ?", "%jin%").Find(&users)
//// SELECT * FROM users WHERE name LIKE "%jin%";

// Multiple Conditions
db.Where("name = ? and age >= ?", "jinzhu", "22").Find(&users)
//// SELECT * FROM users WHERE name = 'jinzhu' AND age >= 22;
```

### Query With Where (Struct & Map)

```go
// Search with struct
db.Where(&User{Name: "jinzhu", Age: 20}).First(&user)
//// SELECT * FROM users WHERE name = "jinzhu" AND age = 20 LIMIT 1;

// Search with map
db.Where(map[string]interface{}{"name": "jinzhu", "age": 20}).Find(&users)
//// SELECT * FROM users WHERE name = "jinzhu" AND age = 20;

// IN For Primary Key
db.Where([]int64{20, 21, 22}).Find(&users)
//// SELECT * FROM users WHERE id IN (20, 21, 22);
```

### Query With Not

```go
// Attribute Not Equal
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

### Inline Search

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

### Query With Or

```
db.Where("role = ?", "admin").Or("role = ?", "super_admin").Find(&users)
//// SELECT * FROM users WHERE role = 'admin' OR role = 'super_admin';

// Or With Struct
db.Where("name = 'jinzhu'").Or(User{Name: "jinzhu 2"}).Find(&users)
//// SELECT * FROM users WHERE name = 'jinzhu' OR name = 'jinzhu 2';

// Or With Map
db.Where("name = 'jinzhu'").Or(map[string]interface{}{"name": "jinzhu 2"}).Find(&users)
```

### Query Chains

Gorm has a chainable API, so you could write query in chain

```go
db.Where("name <> ?", "jinzhu").Where("age >= ? and role <> ?", 20, "admin").Find(&users)
//// SELECT * FROM users WHERE name <> 'jinzhu' AND age >= 20 AND role <> 'admin';

db.Where("role = ?", "admin").Or("role = ?", "super_admin").Not("name = ?", "jinzhu").Find(&users)
```

## Update

### Update an existing struct

```go
user.Name = "jinzhu 2"
user.Age = 100
db.Save(&user)
//// UPDATE users SET name='jinzhu 2', age=100 WHERE id=111;
```

### Update one attribute with `Update`

```
// Update an existing struct's name if name is different
db.Model(&user).Update("name", "hello")
//// UPDATE users SET name='hello' WHERE id=111;

// Find out a struct, and update it if name is different
db.First(&user, 111).Update("name", "hello")
//// SELECT * FROM users LIMIT 1;
//// UPDATE users SET name='hello' WHERE id=111;

// Update a record
db.Table("users").Where(10).Update("name", "hello")
//// UPDATE users SET name='hello' WHERE id = 10;
```

### Update multiple attributes with `Updates`

```
// Update an existing record if have any different attributes
db.Model(&user).Updates(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18 WHERE id = 111;

// Update with Map
db.Table("users").Where(10).Updates(map[string]interface{}{"name": "hello", "age": 18})
//// UPDATE users SET name='hello', age=18 WHERE id = 10;

// Update with Struct
db.Model(User{}).Updates(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18;
```

## Delete

### Delete an existing struct

```go
db.Delete(&email)
// DELETE from emails where id=10;
```

### Batch Delete with search

```go
db.Where("email LIKE ?", "%jinzhu%").Delete(Email{})
// DELETE from emails where email LIKE "%jinhu%";
```

### Soft Delete

If a struct have DeletedAt field, it will get soft delete ability automatically!
For those don't have the filed, will be deleted from database permanently

```go
db.Delete(&user)
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE id = 111;

// Batch delete when search
db.Where("age = ?", 20).Delete(&User{})
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE age = 20;

// For structs have DeletedAt field, when do query, will ignore deleted records by default
db.Where("age = 20").Find(&user)
//// SELECT * FROM users WHERE age = 100 AND (deleted_at IS NULL AND deleted_at <= '0001-01-02');

// Find out all records including those deleted with Unscoped
db.Unscoped().Where("age = 20").Find(&users)
//// SELECT * FROM users WHERE age = 20;

// Permanently delete a record with Unscoped
db.Unscoped().Delete(&order)
// DELETE FROM orders WHERE id=10;
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

Attr's arguments would be used to initialize struct if no record found, but won't be used for search

```go
db.Where(User{Name: "non_existing"}).Attrs(User{Age: 20}).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = 'non_existing';
//// User{Name: "non_existing", Age: 20}

// Above code could be simplified if has only one attribute
db.Where(User{Name: "noexisting_user"}).Attrs("age", 20).FirstOrInit(&user)

// If a record found, Attrs would be just ignored
db.Where(User{Name: "Jinzhu"}).Attrs(User{Age: 30}).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = jinzhu';
//// User{Id: 111, Name: "Jinzhu", Age: 20}

### FirstOrInit With Assign

Assign's arguments would be used to set the struct even a record found, but won't be used for search

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
//// user -> User{Id: 111, Name: "jinzhu", Age: 20}
```

### FirstOrCreate With Attrs

Attr's arguments would be used to initialize struct if no record found, but won't be used for search

```go
db.Where(User{Name: "non_existing"}).Attrs(User{Age: 20}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE name = 'non_existing';
//// User{Id: 112, Name: "non_existing", Age: 20}

db.Where(User{Name: "jinzhu"}).Attrs(User{Age: 30}).FirstOrCreate(&user)
//// User{Id: 111, Name: "jinzhu", Age: 20}
```

### FirstOrCreate With Assign

Assign's arguments would be used to initialize the struct if not record found,
If any record found, will assign those values to the record, and save it back to database.

```go
db.Where(User{Name: "non_existing"}).Assign(User{Age: 20}).FirstOrCreate(&user)
//// user -> User{Id: 112, Name: "non_existing", Age: 20}

db.Where(User{Name: "jinzhu"}).Assign(User{Age: 30}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE name = 'jinzhu';
//// UPDATE users SET age=30 WHERE id = 111;
//// User{Id: 111, Name: "jinzhu", Age: 30}
```

## Select

```go
db.Select("name, age").Find(&users)
//// SELECT name, age FROM users;
```

## Order

```go
db.Order("age desc, name").Find(&users)
//// SELECT * FROM users ORDER BY age desc, name;

db.Order("age desc").Order("name").Find(&users)
//// SELECT * FROM users ORDER BY age desc, name;

// ReOrder
db.Order("age desc").Find(&users1).Order("age", true).Find(&users2)
//// SELECT * FROM users ORDER BY age desc; (users1)
//// SELECT * FROM users ORDER BY age; (users2)
```

## Limit

```go
db.Limit(3).Find(&users)
//// SELECT * FROM users LIMIT 3;

// Cleanup limit with -1
db.Limit(10).Find(&users1).Limit(-1).Find(&users2)
//// SELECT * FROM users LIMIT 10; (users1)
//// SELECT * FROM users; (users2)
```

## Offset

```go
db.Offset(3).Find(&users)
//// SELECT * FROM users OFFSET 3;

// Cleanup offset with -1
db.Offset(10).Find(&users1).Offset(-1).Find(&users2)
//// SELECT * FROM users OFFSET 10; (users1)
//// SELECT * FROM users; (users2)
```

## Count

```go
db.Where("name = ?", "jinzhu").Or("name = ?", "jinzhu 2").Find(&users).Count(&count)
//// SELECT * from USERS WHERE name = 'jinzhu' OR name = 'jinzhu 2'; (users)
//// SELECT count(*) FROM users WHERE name = 'jinzhu' OR name = 'jinzhu 2'; (count)

// Set table name with Model
db.Model(User{}).Where("name = ?", "jinzhu").Count(&count)
//// SELECT count(*) FROM users WHERE name = 'jinzhu' OR name = 'jinzhu 2'; (count)

// Set table name with Table
db.Table("deleted_users").Count(&count)
//// SELECT count(*) FROM deleted_users;
```

## Pluck

Get struct's attribute as map

```go
var ages []int64
db.Find(&users).Pluck("age", &ages)

// Set Table With Model
var names []string
db.Model(&User{}).Pluck("name", &names)
//// SELECT name FROM users;

// Set Table With Table
db.Table("deleted_users").Pluck("name", &names)
//// SELECT name FROM deleted_users;
```

## Callbacks

Callback is a function defined to a struct, the function would be run when reflect a struct to database.
If the function return an error, will prevent following operations. (for example, stop inserting, updating)

Those callbacks are defined now:

`BeforeCreate`, `AfterCreate`
`BeforeUpdate`, `AfterUpdate`
`BeforeSave`, `AfterSave`
`BeforeDelete`, `AfterDelete`

```go
func (u *User) BeforeUpdate() (err error) {
    if u.readonly() {
        err = errors.New("Read Only User")
    }
    return
}
```

## Specify Table Name

```
// When Create Table from struct
db.Table("deleted_users").CreateTable(&User{})

// When Pluck
db.Table("users").Pluck("age", &ages)
//// SELECT age FROM users;

// When Query
var deleted_users []User
db.Table("deleted_users").Find(&deleted_users)
//// SELECT * FROM deleted_users;

// When Delete
db.Table("deleted_users").Where("name = ?", "jinzhu").Delete()
//// DELETE FROM deleted_users WHERE name = 'jinzhu';
```

## Run Raw SQl

```go
db.Exec("drop table users;")
```

## Error Handling

```go
query := db.Where("name = ?", "jinzhu").First(&user)
query := db.First(&user).Limit(10).Find(&users)
//// query.Error keep the latest error happened
//// query.Errors keep all errors happened
//// If an error happened, gorm will stop do query, insert, update, delete

// I often use below code to do error handling in real applicatoins
err = db.Where("name = ?", "jinzhu").First(&user).Error
```

## Advanced Usage With Query Chain

Already excited about above usage? Let's see some magic!

```go
db.First(&first_article).Count(&total_count).Limit(10).Find(&first_page_articles).Offset(10).Find(&second_page_articles)
//// SELECT * FROM articles LIMIT 1; (first_article)
//// SELECT count(*) FROM articles; (count)
//// SELECT * FROM articles LIMIT 10; (first_page_articles)
//// SELECT * FROM articles LIMIT 10 OFFSET 10; (second_page_articles)

db.Where("created_at > ?", "2013-10-10").Find(&cancelled_orders, "state = ?", "cancelled").Find(&shipped_orders, "state = ?", "shipped")
//// SELECT * FROM orders WHERE created_at > '2013/10/10' AND state = 'cancelled'; (cancelled_orders)
//// SELECT * FROM orders WHERE created_at > '2013/10/10' AND state = 'shipped'; (shipped_orders)

db.Where("product_name = ?", "fancy_product").Find(&orders).Find(&shopping_carts)
//// SELECT * FROM orders WHERE product_name = 'fancy_product'; (orders)
//// SELECT * FROM carts WHERE product_name = 'fancy_product'; (shopping_carts)
// Do you noticed the table is different?

db.Where("mail_type = ?", "TEXT").Find(&users1).Table("deleted_users").First(&user2)
//// SELECT * FROM users WHERE mail_type = 'TEXT'; (users1)
//// SELECT * FROM deleted_users WHERE mail_type = 'TEXT'; (users2)

db.Where("email = ?", "x@example.org"').Attrs(User{FromIp: "111.111.111.111"}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE email = 'x@example.org';
//// INSERT INTO "users" (email,from_ip) VALUES ("x@example.org", "111.111.111.111") (if no record found)

// Open your mind, add more cool examples
```

## TODO
* Index, Unique, Valiations
* Auto Migration
* SQL Log
* SQL Query with goroutines
* Only tested with postgres, confirm works with other database adaptors

# Author

**jinzhu**

* <http://github.com/jinzhu>
* <wosmvp@gmail.com>
* <http://twitter.com/zhangjinzhu>
