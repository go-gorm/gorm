# GORM

The fantastic ORM library for Golang, aims to be developer friendly.

## Install

```
go get github.com/jinzhu/gorm
```

## Overview

* Chainable API
* Relations
* Callbacks (before/after create/save/update/delete/find)
* Soft Deletes
* Auto Migrations
* Transactions
* Logger Support
* Bind struct with tag
* Iteration Support via [Rows](#row--rows)
* Scopes
* sql.Scanner support
* Every feature comes with tests
* Convention Over Configuration
* Developer Friendly

## Conventions

* Table name is the plural of struct name's snake case.
  Disable pluralization with `db.SingularTable(true)`, or [Specifying The Table Name For A Struct Permanently With TableName](#specifying-the-table-name-for-a-struct-permanently-with-tablename)
* Column name is the snake case of field's name.
* Use `Id int64` field as primary key.
* Use tag `sql` to change field's property, change the tag name with `db.SetTagIdentifier(new_name)`.
* Use `CreatedAt` to store record's created time if field exists.
* Use `UpdatedAt` to store record's updated time if field exists.
* Use `DeletedAt` to store record's deleted time if field exists. [Soft Delete](#soft-delete)
* Gorm uses reflection to know which tables to work with:

```go
// E.g Finding an existing User
var user User
// Gorm will now know to use table "users" ("user" if pluralisation has been disabled) for all operations.
db.First(&user)

// E.g creating a new User
DB.Save(&User{Name: "xxx"}) // table "users"
```

## Existing Schema

If you have an existing database schema and some of your tables do not follow the conventions, (and you can't rename your table names), please use: [Specifying The Table Name For A Struct Permanently With TableName](#specifying-the-table-name-for-a-struct-permanently-with-tableName).

If your primary key field is different from `id`, you can add a tag to the field structure to specify that this field is a primary key.

```go
type Animal struct { // animals
    AnimalId     int64 `primaryKey:"yes"`
    Birthday     time.Time
    Age          int64
}
```

# Getting Started

```go
import (
    "database/sql"
    "time"
)

type User struct {
    Id           int64
    Birthday     time.Time
    Age          int64
    Name         string  `sql:"size:255"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
    DeletedAt    time.Time

    Emails            []Email         // Embedded structs
    BillingAddress    Address         // Embedded struct
    BillingAddressId  sql.NullInt64   // BillingAddress's foreign key
    ShippingAddress   Address         // Another Embedded struct with same type
    ShippingAddressId int64           // ShippingAddress's foreign key
    IgnoreMe          int64 `sql:"-"` // Ignore this field
}

type Email struct {
    Id         int64
    UserId     int64   // Foreign key for User
    Email      string  `sql:"type:varchar(100);"` // Set this field's type
    Subscribed bool
}

type Address struct {
    Id       int64
    Address1 string         `sql:"not null;unique"` // Set this field as not nullable and unique in database
    Address2 string         `sql:"type:varchar(100);unique"`
    Post     sql.NullString `sql:not null`
    // FYI, "NOT NULL" will only work well with NullXXX Scanner, because golang will initalize a default value for most type...
}
```

## Opening a Database

```go

import "github.com/jinzhu/gorm"
import _ "github.com/lib/pq"
// import _ "github.com/go-sql-driver/mysql"
// import _ "github.com/mattn/go-sqlite3"

db, err := gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
// db, err = gorm.Open("mysql", "gorm:gorm@/gorm?charset=utf8&parseTime=True")
// db, err = gorm.Open("sqlite3", "/tmp/gorm.db")

// Get database connection handle [*sql.DB](http://golang.org/pkg/database/sql/#DB)
d := db.DB()

// With it you could use package `database/sql`'s builtin methods
db.DB().SetMaxIdleConns(10)
db.DB().SetMaxOpenConns(100)
db.DB().Ping()

// By default, table name is plural of struct type, you can use struct type as table name with:
db.SingularTable(true)
```

Gorm is goroutines friendly, so you can create a global variable to keep the connection and use it everywhere in your project.
```go
// db.go
package db

import (
    "fmt"
    "github.com/jinzhu/gorm"
    _ "github.com/lib/pq"
)

var DB gorm.DB
func init() {
    var err error
    DB, err = gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")

    // Connection string parameters for Postgres - http://godoc.org/github.com/lib/pq, if you are using another
    // database refer to the relevant driver's documentation.

    // * dbname - The name of the database to connect to
    // * user - The user to sign in as
    // * password - The user's password
    // * host - The host to connect to. Values that start with / are for unix domain sockets.
    //   (default is localhost)
    // * port - The port to bind to. (default is 5432)
    // * sslmode - Whether or not to use SSL (default is require, this is not the default for libpq)
    //   Valid SSL modes:
    //    * disable - No SSL
    //    * require - Always SSL (skip verification)
    //    * verify-full - Always SSL (require verification)

    if err != nil {
        panic(fmt.Sprintf("Got error when connect database, the error is '%v'", err))
    }
}

// user.go
package user
import . "db"
...
DB.Save(&User{Name: "xxx"})
...
```

## Struct & Database Mapping

```go
// Create table from struct
db.CreateTable(User{})

// Drop table
db.DropTable(User{})
```

### Automating Migrations

Feel free to update your struct, AutoMigrate will keep your database up-to-date.

FYI, AutoMigrate will only add new columns, it won't change the current columns' types or delete unused columns, to make sure your data is safe.

If the table doesn't exist when AutoMigrate is called, gorm will create the table automatically.
(the database first needs to be created manually though...).

```go
db.AutoMigrate(User{})
```

# Gorm API

## Create

```go
user := User{Name: "jinzhu", Age: 18, Birthday: time.Now()}
db.Save(&user)
```

### NewRecord

Returns true if object hasn’t been saved yet (`Id` is blank)

```go
user := User{Name: "jinzhu", Age: 18, Birthday: time.Now()}
db.NewRecord(user) // => true

db.Save(&user)
db.NewRecord(user) // => false
```

### Create With SubStruct

Refer to [Query With Related](#query-with-related) for how to find associations

```go
user := User{
        Name:            "jinzhu",
        BillingAddress:  Address{Address1: "Billing Address - Address 1"},
        ShippingAddress: Address{Address1: "Shipping Address - Address 1"},
        Emails:          []Email{{Email: "jinzhu@example.com"}, {Email: "jinzhu-2@example@example.com"}},
}

db.Save(&user)
//// BEGIN TRANSACTION;
//// INSERT INTO "addresses" (address1) VALUES ("Billing Address - Address 1");
//// INSERT INTO "addresses" (address1) VALUES ("Shipping Address - Address 1");
//// INSERT INTO "users" (name,billing_address_id,shipping_address_id) VALUES ("jinzhu", 1, 2);
//// INSERT INTO "emails" (user_id,email) VALUES (111, "jinzhu@example.com");
//// INSERT INTO "emails" (user_id,email) VALUES (111, "jinzhu-2@example.com");
//// COMMIT;
```

### Create With Predefined Primary key

```go
db.Create(&User{Id: 999, Name: "user 999"})
```

## Query

```go
// Get the first record
db.First(&user)
//// SELECT * FROM users ORDER BY id LIMIT 1;
// Search table `users` is guessed from struct's type

// Get the last record
db.Last(&user)
//// SELECT * FROM users ORDER BY id DESC LIMIT 1;

// Get all records
db.Find(&users)
//// SELECT * FROM users;

// Get record with primary key
db.First(&user, 10)
//// SELECT * FROM users WHERE id = 10;
```

### Query With Where (SQL)

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
db.Where("name in (?)", []string{"jinzhu", "jinzhu 2"}).Find(&users)
//// SELECT * FROM users WHERE name IN ('jinzhu', 'jinzhu 2');

// LIKE
db.Where("name LIKE ?", "%jin%").Find(&users)
//// SELECT * FROM users WHERE name LIKE "%jin%";

// Multiple conditions
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

// IN for primary keys
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

// Not In for primary keys
db.Not([]int64{1,2,3}).First(&user)
//// SELECT * FROM users WHERE id NOT IN (1,2,3);

db.Not([]int64{}).First(&user)
//// SELECT * FROM users;

// SQL string
db.Not("name = ?", "jinzhu").First(&user)
//// SELECT * FROM users WHERE NOT(name = "jinzhu");

// Not with struct
db.Not(User{Name: "jinzhu"}).First(&user)
//// SELECT * FROM users WHERE name <> "jinzhu";
```

### Query With Inline Condition

```go
// Find with primary key
db.First(&user, 23)
//// SELECT * FROM users WHERE id = 23 LIMIT 1;

// SQL string
db.Find(&user, "name = ?", "jinzhu")
//// SELECT * FROM users WHERE name = "jinzhu";

// Multiple conditions
db.Find(&users, "name <> ? and age > ?", "jinzhu", 20)
//// SELECT * FROM users WHERE name <> "jinzhu" AND age > 20;

// Inline search with struct
db.Find(&users, User{Age: 20})
//// SELECT * FROM users WHERE age = 20;

// Inline search with map
db.Find(&users, map[string]interface{}{"age": 20})
//// SELECT * FROM users WHERE age = 20;
```

### Query With Or

```go
db.Where("role = ?", "admin").Or("role = ?", "super_admin").Find(&users)
//// SELECT * FROM users WHERE role = 'admin' OR role = 'super_admin';

// Or With Struct
db.Where("name = 'jinzhu'").Or(User{Name: "jinzhu 2"}).Find(&users)
//// SELECT * FROM users WHERE name = 'jinzhu' OR name = 'jinzhu 2';

// Or With Map
db.Where("name = 'jinzhu'").Or(map[string]interface{}{"name": "jinzhu 2"}).Find(&users)
```

### Query With Related

```go
// Find user's emails with guessed foreign key
db.Model(&user).Related(&emails)
//// SELECT * FROM emails WHERE user_id = 111;

// Find user's billing address with specified foreign key 'BillingAddressId'
db.Model(&user).Related(&address1, "BillingAddressId")
//// SELECT * FROM addresses WHERE id = 123; // 123 is user's BillingAddressId

// Find user with guessed primary key value from email
db.Model(&email).Related(&user)
//// SELECT * FROM users WHERE id = 111; // 111 is email's UserId
```

### Query Chains

Gorm has a chainable API, so you could query like this

```go
db.Where("name <> ?","jinzhu").Where("age >= ? and role <> ?",20,"admin").Find(&users)
//// SELECT * FROM users WHERE name <> 'jinzhu' AND age >= 20 AND role <> 'admin';

db.Where("role = ?", "admin").Or("role = ?", "super_admin").Not("name = ?", "jinzhu").Find(&users)
```

## Update

### Update An Existing Struct

```go
user.Name = "jinzhu 2"
user.Age = 100
db.Save(&user)
//// UPDATE users SET name='jinzhu 2', age=100, updated_at = '2013-11-17 21:34:10' WHERE id=111;
```

### Update One Attribute With `Update`

```go
// Update existing user's name if it is changed
db.Model(&user).Update("name", "hello")
//// UPDATE users SET name='hello', updated_at = '2013-11-17 21:34:10' WHERE id=111;

// Find out a user, and update the name if it is changed
db.First(&user, 111).Update("name", "hello")
//// SELECT * FROM users LIMIT 1;
//// UPDATE users SET name='hello', updated_at = '2013-11-17 21:34:10' WHERE id=111;

// Update name with search condiation and specified table name
db.Table("users").Where(10).Update("name", "hello")
//// UPDATE users SET name='hello' WHERE id = 10;
```

### Update Multiple Attributes With `Updates`

```go
// Update user's name and age if they are changed
db.Model(&user).Updates(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18, updated_at = '2013-11-17 21:34:10' WHERE id = 111;

// Updates with Map
db.Table("users").Where(10).Updates(map[string]interface{}{"name": "hello", "age": 18})
//// UPDATE users SET name='hello', age=18 WHERE id = 10;

// Updates with Struct
db.Model(User{}).Updates(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18;
```

### Update Attributes Without Callbacks

```go
db.Model(&user).UpdateColumn("name", "hello")
//// UPDATE users SET name='hello' WHERE id = 111;

db.Model(&user).UpdateColumns(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18 WHERE id = 111;
```

### Get Affected Records Count

```go
db.Model(User{}).Updates(User{Name: "hello", Age: 18}).RowsAffected
```
## Delete

### Delete An Existing Struct

```go
db.Delete(&email)
// DELETE from emails where id=10;
```

### Batch Delete With Search

```go
db.Where("email LIKE ?", "%jinzhu%").Delete(Email{})
// DELETE from emails where email LIKE "%jinhu%";
```

### Soft Delete

If a struct has a DeletedAt field, it will get a soft delete ability automatically!

Structs that don't have a DeletedAt field will be deleted from the database permanently.

```go
db.Delete(&user)
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE id = 111;

// Delete with search condition
db.Where("age = ?", 20).Delete(&User{})
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE age = 20;

// Soft deleted records will be ignored when searched
db.Where("age = 20").Find(&user)
//// SELECT * FROM users WHERE age = 100 AND (deleted_at IS NULL AND deleted_at <= '0001-01-02');

// Find soft deleted records with Unscoped
db.Unscoped().Where("age = 20").Find(&users)
//// SELECT * FROM users WHERE age = 20;

// Delete record permanently with Unscoped
db.Unscoped().Delete(&order)
// DELETE FROM orders WHERE id=10;
```

## FirstOrInit

Try to get the first record, if failed, will initialize the struct with search conditions.

(only supports search conditions map and struct)

```go
db.FirstOrInit(&user, User{Name: "non_existing"})
//// User{Name: "non_existing"}

db.Where(User{Name: "Jinzhu"}).FirstOrInit(&user)
//// User{Id: 111, Name: "Jinzhu", Age: 20}

db.FirstOrInit(&user, map[string]interface{}{"name": "jinzhu"})
//// User{Id: 111, Name: "Jinzhu", Age: 20}
```

### FirstOrInit With Attrs

Ignore Attrs's arguments when searching, but use them to initialize the struct if no record is found.

```go
db.Where(User{Name: "non_existing"}).Attrs(User{Age: 20}).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = 'non_existing';
//// User{Name: "non_existing", Age: 20}

// Or write it like this if has only one attribute
db.Where(User{Name: "noexisting_user"}).Attrs("age", 20).FirstOrInit(&user)

// If a record found, Attrs would be ignored
db.Where(User{Name: "Jinzhu"}).Attrs(User{Age: 30}).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = jinzhu';
//// User{Id: 111, Name: "Jinzhu", Age: 20}
```

### FirstOrInit With Assign

Ignore Assign's arguments when searching, but use them to fill the struct regardless, whether the record is found or not.

```go
db.Where(User{Name: "non_existing"}).Assign(User{Age: 20}).FirstOrInit(&user)
//// User{Name: "non_existing", Age: 20}

db.Where(User{Name: "Jinzhu"}).Assign(User{Age: 30}).FirstOrInit(&user)
//// User{Id: 111, Name: "Jinzhu", Age: 30}
```

## FirstOrCreate

Try to get the first record, if failed, will initialize the struct with the search conditions and insert it in the database.

```go
db.FirstOrCreate(&user, User{Name: "non_existing"})
//// User{Id: 112, Name: "non_existing"}

db.Where(User{Name: "Jinzhu"}).FirstOrCreate(&user)
//// User{Id: 111, Name: "Jinzhu"}

db.FirstOrCreate(&user, map[string]interface{}{"name": "jinzhu", "age": 30})
//// user -> User{Id: 111, Name: "jinzhu", Age: 20}
```

### FirstOrCreate With Attrs

Ignore Attrs's arguments when searching, but use them to initialize the struct if no record is found.

```go
db.Where(User{Name: "non_existing"}).Attrs(User{Age: 20}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE name = 'non_existing';
//// User{Id: 112, Name: "non_existing", Age: 20}

db.Where(User{Name: "jinzhu"}).Attrs(User{Age: 30}).FirstOrCreate(&user)
//// User{Id: 111, Name: "jinzhu", Age: 20}
```

### FirstOrCreate With Assign

Ignore Assign's arguments when searching, but use them to fill the struct regardless, whether the record is found or not, then save it back to the database.

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

// Multiple orders
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

// Remove limit with -1
db.Limit(10).Find(&users1).Limit(-1).Find(&users2)
//// SELECT * FROM users LIMIT 10; (users1)
//// SELECT * FROM users; (users2)
```

## Offset

```go
db.Offset(3).Find(&users)
//// SELECT * FROM users OFFSET 3;

// Remove offset with -1
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

Get struct's selected attributes as a map.

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

// Plucking more than one column? Do it like this:
db.Select("name, age").Find(&users)
```

## Transactions
All individual save and delete operations are run in a transaction by default.

```go
tx := db.Begin()

user := User{Name: "transcation"}

tx.Save(&u)
tx.Update("age": 90)
// do whatever

// rollback
tx.Rollback()

// commit
tx.Commit()
```

## Callbacks

Callbacks are methods defined on the struct's pointer.
If any callback returns an error, gorm will stop future operations and rollback all changes.

Here is a list of all available callbacks,
listed in the same order in which they will get called during the respective operations.

### Creating An Object

```go
BeforeSave
BeforeCreate
// save before associations
// save self
// save after associations
AfterCreate
AfterSave
```
### Updating An Object

```go
BeforeSave
BeforeUpdate
// save before associations
// save self
// save after associations
AfterUpdate
AfterSave
```

### Destroying An Object

```go
BeforeDelete
// delete self
AfterDelete
```

### After Find

```go
// load record/records from database
AfterFind
```

Here is an example:

```go
func (u *User) BeforeUpdate() (err error) {
    if u.readonly() {
        err = errors.New("Read Only User!")
    }
    return
}

// Rollback the insertion if there are more than 1000 users (hypothetical example)
func (u *User) AfterCreate() (err error) {
    if (u.Id > 1000) { // Just as an example, don't use Id to count users!
        err = errors.New("Only 1000 users allowed!")
    }
    return
}
```

```go
// As you know, the save/delete operations are running in a transaction
// This is means that all your changes will be rolled back if there are any errors
// If you want your changes in callbacks be run in the same transaction
// you have to pass the transaction as argument to the function
func (u *User) AfterCreate(tx *gorm.DB) (err error) {
    tx.Model(u).Update("role", "admin")
    return
}
```

## Specifying The Table Name

```go
// Create `deleted_users` table with User's fields
db.Table("deleted_users").CreateTable(&User{})

// Search from table `deleted_users`
var deleted_users []User
db.Table("deleted_users").Find(&deleted_users)
//// SELECT * FROM deleted_users;

// Delete results from table `deleted_users` with search conditions
db.Table("deleted_users").Where("name = ?", "jinzhu").Delete()
//// DELETE FROM deleted_users WHERE name = 'jinzhu';
```

### Specifying The Table Name For A Struct Permanently with TableName

```go
type Cart struct {
}

func (c Cart) TableName() string {
    return "shopping_cart"
}

func (u User) TableName() string {
    if u.Role == "admin" {
        return "admin_users"
    } else {
        return "users"
    }
}
```

## Scopes

```go
func AmountGreaterThan1000(d *gorm.DB) *gorm.DB {
  d.Where("amount > ?", 1000)
}

func PaidWithCreditCard(d *gorm.DB) *gorm.DB {
  d.Where("pay_mode_sign = ?", "C")
}

func PaidWithCod(d *gorm.DB) *gorm.DB {
  d.Where("pay_mode_sign = ?", "C")
}

func OrderStatus(status []string) func (d *gorm.DB) *gorm.DB {
  return func (d *gorm.DB) *gorm.DB {
     return d.Scopes(AmountGreaterThan1000).Where("status in (?)", status)
  }
}

db.Scopes(AmountGreaterThan1000, PaidWithCreditCard).Find(&orders)
// Find all credit card orders and amount greater than 1000

db.Scopes(AmountGreaterThan1000, PaidWithCod).Find(&orders)
// Find all COD orders and amount greater than 1000

db.Scopes(OrderStatus([]string{"paid", "shipped"})).Find(&orders)
// Find all paid, shipped orders and amount greater than 1000
```

## Logger

Gorm has built-in logger support, enable it with:

```go
db.LogMode(true)
```

![logger](https://raw.github.com/jinzhu/gorm/master/images/logger.png)

```go
// Use your own logger
// Refer to gorm's default logger for how to format messages: https://github.com/jinzhu/gorm/blob/master/logger.go#files
db.SetLogger(log.New(os.Stdout, "\r\n", 0))

// If you want to use gorm's default log format, then you could just do it like this
db.SetLogger(gorm.Logger{revel.TRACE})

// Disable logging
db.LogMode(false)

// Enable logging for a single operation, to make debugging easy
db.Debug().Where("name = ?", "jinzhu").First(&User{})
```

## Row & Rows

Row & Rows is not chainable, it works just like `QueryRow` and `Query`.

```go
row := db.Table("users").Where("name = ?", "jinzhu").Select("name, age").Row() // (*sql.Row)
row.Scan(&name, &age)

rows, err := db.Model(User{}).Where("name = ?", "jinzhu").Select("name, age, email").Rows() // (*sql.Rows, error)
defer rows.Close()
for rows.Next() {
  ...
  rows.Scan(&name, &age, &email)
  ...
}

// Rows() with raw sql
rows, err := db.Raw("select name, age, email from users where name = ?", "jinzhu").Rows() // (*sql.Rows, error)
defer rows.Close()
for rows.Next() {
  ...
  rows.Scan(&name, &age, &email)
  ...
}
```

## Scan

Scan sql results into a struct.

```go
type Result struct {
	Name string
	Age  int
}

var result Result
db.Table("users").Select("name, age").Where("name = ?", 3).Scan(&result)

// Scan raw sql
db.Raw("SELECT name, age FROM users WHERE name = ?", 3).Scan(&result)
```

## Group & Having

```go
rows, err := db.Table("orders").Select("date(created_at) as date, sum(amount) as total").Group("date(created_at)").Rows()
for rows.Next() {
  ...
}

rows, err := db.Table("orders").Select("date(created_at) as date, sum(amount) as total").Group("date(created_at)").Having("sum(amount) > ?", 100).Rows()
for rows.Next() {
  ...
}

type Result struct {
	Date  time.Time
	Total int64
}
db.Table("orders").Select("date(created_at) as date, sum(amount) as total").Group("date(created_at)").Having("sum(amount) > ?", 100).Scan(&results)
```

## Joins

```go
rows, err := db.Table("users").Select("users.name, emails.email").Joins("left join emails on emails.user_id = users.id").Rows()
for rows.Next() {
  ...
}

db.Table("users").Select("users.name, emails.email").Joins("left join emails on emails.user_id = users.id").Scan(&results)
```

## Indices

```go
// single column index
db.Model(User{}).AddIndex("idx_user_name", "name")

// multiple column index
db.Model(User{}).AddIndex("idx_user_name_age", "name", "age")

// single column unique index
db.Model(User{}).AddUniqueIndex("idx_user_name", "name")

// multiple column unique index
db.Model(User{}).AddUniqueIndex("idx_user_name_age", "name", "age")

// remove index
db.Model(User{}).RemoveIndex("idx_user_name")
```

## Run Raw SQL

```go
// Raw SQL
db.Exec("DROP TABLE users;")

// Raw SQL with arguments
db.Exec("UPDATE orders SET shipped_at=? WHERE id IN (?)", time.Now, []int64{11,22,33})
```

## Error Handling

```go
query := db.Where("name = ?", "jinzhu").First(&user)
query := db.First(&user).Limit(10).Find(&users)
//// query.Error returns the last error
//// query.Errors returns all errors that have taken place
//// If an error has taken place, gorm will stop all following operations

// I often use some code like below to do error handling when writing applications
if err := db.Where("name = ?", "jinzhu").First(&user).Error; err != nil {
  // ...
}

// If no record is found, gorm will return RecordNotFound error, you could check it with
db.Where("name = ?", "hello world").First(&User{}).Error == gorm.RecordNotFound

// Or use the shortcut method
if db.Where("name = ?", "hello world").First(&user).RecordNotFound() {
  panic("no record found")
} else {
  user.Blalala()
}

if db.Model(&user).Related(&credit_card).RecordNotFound() {
  panic("no credit card found")
}
```

## Advanced Usage With Query Chaining

Already excited with what gorm has to offer? Let's see some magic!

```go
db.First(&first_article).Count(&total_count).Limit(10).Find(&first_page_articles).Offset(10).Find(&second_page_articles)
//// SELECT * FROM articles LIMIT 1; (first_article)
//// SELECT count(*) FROM articles; (total_count)
//// SELECT * FROM articles LIMIT 10; (first_page_articles)
//// SELECT * FROM articles LIMIT 10 OFFSET 10; (second_page_articles)


// Mix where conditions with inline conditions
db.Where("created_at > ?", "2013-10-10").Find(&cancelled_orders, "state = ?", "cancelled").Find(&shipped_orders, "state = ?", "shipped")
//// SELECT * FROM orders WHERE created_at > '2013/10/10' AND state = 'cancelled'; (cancelled_orders)
//// SELECT * FROM orders WHERE created_at > '2013/10/10' AND state = 'shipped'; (shipped_orders)


// Use variables to keep query chain
todays_orders := db.Where("created_at > ?", "2013-10-29")
cancelled_orders := todays_orders.Where("state = ?", "cancelled")
shipped_orders := todays_orders.Where("state = ?", "shipped")


// Search with shared conditions from different tables
db.Where("product_name = ?", "fancy_product").Find(&orders).Find(&shopping_carts)
//// SELECT * FROM orders WHERE product_name = 'fancy_product'; (orders)
//// SELECT * FROM carts WHERE product_name = 'fancy_product'; (shopping_carts)


// Search with shared conditions from different tables with specified table
db.Where("mail_type = ?", "TEXT").Find(&users1).Table("deleted_users").Find(&users2)
//// SELECT * FROM users WHERE mail_type = 'TEXT'; (users1)
//// SELECT * FROM deleted_users WHERE mail_type = 'TEXT'; (users2)


// An example on how to use FirstOrCreate
db.Where("email = ?", "x@example.org").Attrs(User{RegisteredIp: "111.111.111.111"}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE email = 'x@example.org';
//// INSERT INTO "users" (email,registered_ip) VALUES ("x@example.org", "111.111.111.111")  // if no record found
```

## TODO
* Support plugin
  BeforeQuery
  BeforeSave
  BeforeCreate
  BeforeUpdate
  BeforeDelete
  AfterQuery
  AfterSave
  AfterCreate
  AfterUpdate
  SoftDelete
      BeforeQuery
      BeforeSave
      BeforeDelete

  db.RegisterPlugin("xxx")
  db.RegisterCallback("BeforeQuery", func() {})
  db.RegisterCallback("BeforeSave", func() {})
  db.RegisterFuncation("Search", func() {})
  db.Model(&[]User{}).Limit(10).Do("Search", "vip", "china")
  db.Mode(&User{}).Do("EditForm").Get("edit_form_html")

  DefaultValue, DefaultTimeZone, R/W Splitting, Validation
* Getter/Setter
  share or not? transaction?
* Github Pages
* Includes
* AlertColumn, DropColumn

# Author

**jinzhu**

* <http://github.com/jinzhu>
* <wosmvp@gmail.com>
* <http://twitter.com/zhangjinzhu>

## License

Released under the [MIT License](http://www.opensource.org/licenses/MIT).

[![GoDoc](https://godoc.org/github.com/jinzhu/gorm?status.png)](http://godoc.org/github.com/jinzhu/gorm)
