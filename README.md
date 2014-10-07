# GORM

The fantastic ORM library for Golang, aims to be developer friendly.

## Overview

* Chainable API
* Embedded Structs
* Relations
* Callbacks (before/after create/save/update/delete/find)
* Soft Deletes
* Auto Migrations
* Transactions
* Customizable Logger
* Iteration Support via [Rows](#row--rows)
* Scopes
* sql.Scanner support
* Every feature comes with tests
* Convention Over Configuration
* Developer Friendly

## Conventions

* Table name is the plural of struct name's snake case, you can disable pluralization with `db.SingularTable(true)`, or [Specifying The Table Name For A Struct Permanently With TableName](#specifying-the-table-name-for-a-struct-permanently-with-tablename)

```go
// E.g finding an existing User
var user User
// Gorm will know to use table "users" ("user" if pluralisation has been disabled) for all operations.
db.First(&user)

// creating a new User
db.Save(&User{Name: "xxx"}) // table "users"
```

* Column name is the snake case of field's name
* Use `Id` field as primary key
* Use tag `sql` to change field's property, change the tag name with `db.SetTagIdentifier(new_name)`
* Use `CreatedAt` to store record's created time if field exists
* Use `UpdatedAt` to store record's updated time if field exists
* Use `DeletedAt` to store record's deleted time if field exists [Soft Delete](#soft-delete)

# Getting Started

## Install

```
go get -u github.com/jinzhu/gorm
```

## Define Models (Structs)

```go
type User struct {
	Id           int64
	Birthday     time.Time
	Age          int64
	Name         string  `sql:"size:255"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    time.Time

	Emails            []Email         // One-To-Many relationship (has many)
	BillingAddress    Address         // One-To-One relationship (has one)
	BillingAddressId  sql.NullInt64   // Foreign key of BillingAddress
	ShippingAddress   Address         // One-To-One relationship (has one)
	ShippingAddressId int64           // Foreign key of ShippingAddress
	IgnoreMe          int64 `sql:"-"` // Ignore this field
	Languages         []Language `gorm:"many2many:user_languages;"` // Many-To-Many relationship, 'user_languages' is join table
}

type Email struct {
	Id         int64
	UserId     int64   // Foreign key for User (belongs to)
	Email      string  `sql:"type:varchar(100);"` // Set field's type
	Subscribed bool
}

type Address struct {
	Id       int64
	Address1 string         `sql:"not null;unique"` // Set field as not nullable and unique
	Address2 string         `sql:"type:varchar(100);unique"`
	Post     sql.NullString `sql:not null`
}

type Language struct {
	Id   int64
	Name string
}
```

## Initialize Database

```go

import (
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

db, err := gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
// db, err := gorm.Open("mysql", "gorm:gorm@/gorm?charset=utf8&parseTime=True")
// db, err := gorm.Open("sqlite3", "/tmp/gorm.db")

// Get database connection handle [*sql.DB](http://golang.org/pkg/database/sql/#DB)
db.DB()

// Then you could invoke `*sql.DB`'s functions with it
db.DB().Ping()
db.DB().SetMaxIdleConns(10)
db.DB().SetMaxOpenConns(100)

// Disable table name's pluralization
db.SingularTable(true)
```

## Migration

```go
// Create table
db.CreateTable(&User{})

// Drop table
db.DropTable(&User{})

// Drop table if exists
db.DropTableIfExists(&User{})

// Automating Migration
db.AutoMigrate(&User{})
db.AutoMigrate(&User{}, &Product{}, &Order{})

// Feel free to change your struct, AutoMigrate will keep your database up-to-date.
// Fyi, AutoMigrate will only *add new columns*, it won't update column's type or delete unused columns, to make sure your data is safe.
// If the table is not existing, AutoMigrate will create the table automatically.

// Add index
db.Model(&User{}).AddIndex("idx_user_name", "name")

// Multiple column index
db.Model(&User{}).AddIndex("idx_user_name_age", "name", "age")

// Add unique index
db.Model(&User{}).AddUniqueIndex("idx_user_name", "name")

// Multiple column unique index
db.Model(&User{}).AddUniqueIndex("idx_user_name_age", "name", "age")

// Remove index
db.Model(&User{}).RemoveIndex("idx_user_name")
```

# Basic CRUD

## Create Record

```go
user := User{Name: "Jinzhu", Age: 18, Birthday: time.Now()}

// returns true if record hasn’t been saved (primary key `Id` is blank)
db.NewRecord(user) // => true

db.Create(&user)

// will ruturn false after `user` created
db.NewRecord(user) // => false

// You could use `Save` to create record also if its primary key is null
db.Save(&user)

// Associations will be saved automatically when insert the record
user := User{
	Name:            "jinzhu",
	BillingAddress:  Address{Address1: "Billing Address - Address 1"},
	ShippingAddress: Address{Address1: "Shipping Address - Address 1"},
	Emails:          []Email{{Email: "jinzhu@example.com"}, {Email: "jinzhu-2@example@example.com"}},
	Languages:       []Language{{Name: "ZH"}, {Name: "EN"}},
}

db.Create(&user)
//// BEGIN TRANSACTION;
//// INSERT INTO "addresses" (address1) VALUES ("Billing Address - Address 1");
//// INSERT INTO "addresses" (address1) VALUES ("Shipping Address - Address 1");
//// INSERT INTO "users" (name,billing_address_id,shipping_address_id) VALUES ("jinzhu", 1, 2);
//// INSERT INTO "emails" (user_id,email) VALUES (111, "jinzhu@example.com");
//// INSERT INTO "emails" (user_id,email) VALUES (111, "jinzhu-2@example.com");
//// INSERT INTO "languages" ("name") VALUES ('ZH');
//// INSERT INTO user_languages ("user_id","language_id") VALUES (111, 1);
//// INSERT INTO "languages" ("name") VALUES ('EN');
//// INSERT INTO user_languages ("user_id","language_id") VALUES (111, 2);
//// COMMIT;
```

Refer [Associations](#associations) for how to work with associations

## Query

```go
// Get the first record
db.First(&user)
//// SELECT * FROM users ORDER BY id LIMIT 1;

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

### Query With Where (Plain SQL)

```go
// Get the first matched record
db.Where("name = ?", "jinzhu").First(&user)
//// SELECT * FROM users WHERE name = 'jinzhu' limit 1;

// Get all matched records
db.Where("name = ?", "jinzhu").Find(&users)
//// SELECT * FROM users WHERE name = 'jinzhu';

db.Where("name <> ?", "jinzhu").Find(&users)

// IN
db.Where("name in (?)", []string{"jinzhu", "jinzhu 2"}).Find(&users)

// LIKE
db.Where("name LIKE ?", "%jin%").Find(&users)

// AND
db.Where("name = ? and age >= ?", "jinzhu", "22").Find(&users)
```

### Query With Where (Struct & Map)

```go
// Struct
db.Where(&User{Name: "jinzhu", Age: 20}).First(&user)
//// SELECT * FROM users WHERE name = "jinzhu" AND age = 20 LIMIT 1;

// Map
db.Where(map[string]interface{}{"name": "jinzhu", "age": 20}).Find(&users)
//// SELECT * FROM users WHERE name = "jinzhu" AND age = 20;

// Slice of primary keys
db.Where([]int64{20, 21, 22}).Find(&users)
//// SELECT * FROM users WHERE id IN (20, 21, 22);
```

### Query With Not

```go
db.Not("name", "jinzhu").First(&user)
//// SELECT * FROM users WHERE name <> "jinzhu" LIMIT 1;

// Not In
db.Not("name", []string{"jinzhu", "jinzhu 2"}).Find(&users)
//// SELECT * FROM users WHERE name NOT IN ("jinzhu", "jinzhu 2");

// Not In slice of primary keys
db.Not([]int64{1,2,3}).First(&user)
//// SELECT * FROM users WHERE id NOT IN (1,2,3);

db.Not([]int64{}).First(&user)
//// SELECT * FROM users;

// Plain SQL
db.Not("name = ?", "jinzhu").First(&user)
//// SELECT * FROM users WHERE NOT(name = "jinzhu");

// Struct
db.Not(User{Name: "jinzhu"}).First(&user)
//// SELECT * FROM users WHERE name <> "jinzhu";
```

### Query With Inline Condition

```go
// Get by primary key
db.First(&user, 23)
//// SELECT * FROM users WHERE id = 23 LIMIT 1;

// Plain SQL
db.Find(&user, "name = ?", "jinzhu")
//// SELECT * FROM users WHERE name = "jinzhu";

db.Find(&users, "name <> ? AND age > ?", "jinzhu", 20)
//// SELECT * FROM users WHERE name <> "jinzhu" AND age > 20;

// Struct
db.Find(&users, User{Age: 20})
//// SELECT * FROM users WHERE age = 20;

// Map
db.Find(&users, map[string]interface{}{"age": 20})
//// SELECT * FROM users WHERE age = 20;
```

### Query With Or

```go
db.Where("role = ?", "admin").Or("role = ?", "super_admin").Find(&users)
//// SELECT * FROM users WHERE role = 'admin' OR role = 'super_admin';

// Struct
db.Where("name = 'jinzhu'").Or(User{Name: "jinzhu 2"}).Find(&users)
//// SELECT * FROM users WHERE name = 'jinzhu' OR name = 'jinzhu 2';

// Map
db.Where("name = 'jinzhu'").Or(map[string]interface{}{"name": "jinzhu 2"}).Find(&users)
```

### Query Chains

Gorm has a chainable API, you could use it like this

```go
db.Where("name <> ?","jinzhu").Where("age >= ? and role <> ?",20,"admin").Find(&users)
//// SELECT * FROM users WHERE name <> 'jinzhu' AND age >= 20 AND role <> 'admin';

db.Where("role = ?", "admin").Or("role = ?", "super_admin").Not("name = ?", "jinzhu").Find(&users)
```

## Update

```go
// Update an existing struct
db.First(&user)
user.Name = "jinzhu 2"
user.Age = 100
db.Save(&user)
//// UPDATE users SET name='jinzhu 2', age=100, updated_at = '2013-11-17 21:34:10' WHERE id=111;

// Update an attribute if it is changed
db.Model(&user).Update("name", "hello")
//// UPDATE users SET name='hello', updated_at = '2013-11-17 21:34:10' WHERE id=111;

db.First(&user, 111).Update("name", "hello")
//// SELECT * FROM users LIMIT 1;
//// UPDATE users SET name='hello', updated_at = '2013-11-17 21:34:10' WHERE id=111;

// Update multiple attributes if they are changed
db.Model(&user).Updates(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18, updated_at = '2013-11-17 21:34:10' WHERE id = 111;
```

### Update Without Callbacks

By default, update will call BeforeUpdate, AfterUpdate callbacks, if you want to update w/o callbacks:

```go
db.Model(&user).UpdateColumn("name", "hello")
//// UPDATE users SET name='hello' WHERE id = 111;

db.Model(&user).UpdateColumns(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18 WHERE id = 111;
```

### Batch Updates

```go
db.Table("users").Where("id = ?", 10).Updates(map[string]interface{}{"name": "hello", "age": 18})
//// UPDATE users SET name='hello', age=18 WHERE id = 10;

db.Model(User{}).Updates(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18;

// Callbacks won't be run when do batch updates

// You may would like to know how many records updated when do batch updates
// You could get it with `RowsAffected`
db.Model(User{}).Updates(User{Name: "hello", Age: 18}).RowsAffected
```

## Delete

```go
// Delete an existing record
db.Delete(&email)
//// DELETE from emails where id=10;
```

### Batch Delete

```go
db.Where("email LIKE ?", "%jinzhu%").Delete(Email{})
//// DELETE from emails where email LIKE "%jinhu%";
```

### Soft Delete

If struct has `DeletedAt` field, it will get soft delete ability automatically!
Then it won't be deleted from database permanently when call `Delete`.

```go
db.Delete(&user)
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE id = 111;

// Batch Delete
db.Where("age = ?", 20).Delete(&User{})
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE age = 20;

// Soft deleted records will be ignored when query them
db.Where("age = 20").Find(&user)
//// SELECT * FROM users WHERE age = 20 AND (deleted_at IS NULL AND deleted_at <= '0001-01-02');

// Find soft deleted records with Unscoped
db.Unscoped().Where("age = 20").Find(&users)
//// SELECT * FROM users WHERE age = 20;

// Delete record permanently with Unscoped
db.Unscoped().Delete(&order)
//// DELETE FROM orders WHERE id=10;
```

## Associations

### Has One

```go
// User has one address
db.Model(&user).Related(&address)
//// SELECT * FROM addresses WHERE id = 123; // 123 is user's foreign key AddressId

// Specify the foreign key
db.Model(&user).Related(&address1, "BillingAddressId")
//// SELECT * FROM addresses WHERE id = 123; // 123 is user's foreign key BillingAddressId
```

### Belongs To

```go
// Email belongs to user
db.Model(&email).Related(&user)
//// SELECT * FROM users WHERE id = 111; // 111 is email's foreign key UserId

// Specify the foreign key
db.Model(&email).Related(&user, "ProfileId")
//// SELECT * FROM users WHERE id = 111; // 111 is email's foreign key ProfileId
```

### Has Many

```go
// User has many emails
db.Model(&user).Related(&emails)
//// SELECT * FROM emails WHERE user_id = 111;
// user_id is the foreign key, 111 is user's primary key's value

// Specify the foreign key
db.Model(&user).Related(&emails, "ProfileId")
//// SELECT * FROM emails WHERE profile_id = 111;
// profile_id is the foreign key, 111 is user's primary key's value
```

### Many To Many

```go
// User has many languages and belongs to many languages
db.Model(&user).Related(&languages, "Languages")
//// SELECT * FROM "languages" INNER JOIN "user_languages" ON "user_languages"."language_id" = "languages"."id" WHERE "user_languages"."user_id" = 111
// `Languages` is user's column name, this column's tag defined join table like this `gorm:"many2many:user_languages;"`
```

There is also a mode used to handle many to many relations easily

```go
// Query
db.Model(&user).Association("Languages").Find(&languages)
// same as `db.Model(&user).Related(&languages, "Languages")`

db.Where("name = ?", "ZH").First(&languageZH)
db.Where("name = ?", "EN").First(&languageEN)

// Append
db.Model(&user).Association("Languages").Append([]Language{languageZH, languageEN})
db.Model(&user).Association("Languages").Append([]Language{{Name: "DE"}})
db.Model(&user).Association("Languages").Append(Language{Name: "DE"})

// Delete
db.Model(&user).Association("Languages").Delete([]Language{languageZH, languageEN})
db.Model(&user).Association("Languages").Delete(languageZH, languageEN)

// Replace
db.Model(&user).Association("Languages").Replace([]Language{languageZH, languageEN})
db.Model(&user).Association("Languages").Replace(Language{Name: "DE"}, languageEN)

// Count
db.Model(&user).Association("Languages").Count()
// Return the count of languages the user has

// Clear
db.Model(&user).Association("Languages").Clear()
// Remove all relations between the user and languages
```

## Advanced Usage

## FirstOrInit

Get the first matched record, or initialize a record with search conditions.

```go
// Unfound
db.FirstOrInit(&user, User{Name: "non_existing"})
//// user -> User{Name: "non_existing"}

// Found
db.Where(User{Name: "Jinzhu"}).FirstOrInit(&user)
//// user -> User{Id: 111, Name: "Jinzhu", Age: 20}
db.FirstOrInit(&user, map[string]interface{}{"name": "jinzhu"})
//// user -> User{Id: 111, Name: "Jinzhu", Age: 20}
```

### Attrs

Ignore some values when searching, but use them to initialize the struct if record is not found.

```go
// Unfound
db.Where(User{Name: "non_existing"}).Attrs(User{Age: 20}).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = 'non_existing';
//// user -> User{Name: "non_existing", Age: 20}

db.Where(User{Name: "noexisting_user"}).Attrs("age", 20).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = 'non_existing';
//// user -> User{Name: "non_existing", Age: 20}

// Found
db.Where(User{Name: "Jinzhu"}).Attrs(User{Age: 30}).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = jinzhu';
//// user -> User{Id: 111, Name: "Jinzhu", Age: 20}
```

### Assign

Ignore some values when searching, but assign it to the result regardless it is found or not.

```go
// Unfound
db.Where(User{Name: "non_existing"}).Assign(User{Age: 20}).FirstOrInit(&user)
//// user -> User{Name: "non_existing", Age: 20}

// Found
db.Where(User{Name: "Jinzhu"}).Assign(User{Age: 30}).FirstOrInit(&user)
//// SELECT * FROM USERS WHERE name = jinzhu';
//// user -> User{Id: 111, Name: "Jinzhu", Age: 30}
```

## FirstOrCreate

Get the first matched record, or create with search conditions.

```go
// Unfound
db.FirstOrCreate(&user, User{Name: "non_existing"})
//// INSERT INTO "users" (name) VALUES ("non_existing");
//// user -> User{Id: 112, Name: "non_existing"}

// Found
db.Where(User{Name: "Jinzhu"}).FirstOrCreate(&user)
//// user -> User{Id: 111, Name: "Jinzhu"}
```

### Attrs

Ignore some values when searching, but use them to create the struct if record is not found. like `FirstOrInit`

```go
// Unfound
db.Where(User{Name: "non_existing"}).Attrs(User{Age: 20}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE name = 'non_existing';
//// INSERT INTO "users" (name, age) VALUES ("non_existing", 20);
//// user -> User{Id: 112, Name: "non_existing", Age: 20}

// Found
db.Where(User{Name: "jinzhu"}).Attrs(User{Age: 30}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE name = 'jinzhu';
//// user -> User{Id: 111, Name: "jinzhu", Age: 20}
```

### Assign

Ignore some values when searching, but assign it to the record regardless it is found or not, then save back to database. like `FirstOrInit`

```go
// Unfound
db.Where(User{Name: "non_existing"}).Assign(User{Age: 20}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE name = 'non_existing';
//// INSERT INTO "users" (name, age) VALUES ("non_existing", 20);
//// user -> User{Id: 112, Name: "non_existing", Age: 20}

// Found
db.Where(User{Name: "jinzhu"}).Assign(User{Age: 30}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE name = 'jinzhu';
//// UPDATE users SET age=30 WHERE id = 111;
//// user -> User{Id: 111, Name: "jinzhu", Age: 30}
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

// Cancel limit condition with -1
db.Limit(10).Find(&users1).Limit(-1).Find(&users2)
//// SELECT * FROM users LIMIT 10; (users1)
//// SELECT * FROM users; (users2)
```

## Offset

```go
db.Offset(3).Find(&users)
//// SELECT * FROM users OFFSET 3;

// Cancel offset condition with -1
db.Offset(10).Find(&users1).Offset(-1).Find(&users2)
//// SELECT * FROM users OFFSET 10; (users1)
//// SELECT * FROM users; (users2)
```

## Count

```go
db.Where("name = ?", "jinzhu").Or("name = ?", "jinzhu 2").Find(&users).Count(&count)
//// SELECT * from USERS WHERE name = 'jinzhu' OR name = 'jinzhu 2'; (users)
//// SELECT count(*) FROM users WHERE name = 'jinzhu' OR name = 'jinzhu 2'; (count)

db.Model(User{}).Where("name = ?", "jinzhu").Count(&count)
//// SELECT count(*) FROM users WHERE name = 'jinzhu'; (count)

db.Table("deleted_users").Count(&count)
//// SELECT count(*) FROM deleted_users;
```

## Pluck

Get selected attributes as map

```go
var ages []int64
db.Find(&users).Pluck("age", &ages)

var names []string
db.Model(&User{}).Pluck("name", &names)

db.Table("deleted_users").Pluck("name", &names)

// Requesting more than one column? Do it like this:
db.Select("name, age").Find(&users)
```

## Raw SQL

```go
db.Exec("DROP TABLE users;")
db.Exec("UPDATE orders SET shipped_at=? WHERE id IN (?)", time.Now, []int64{11,22,33})
```

## Row & Rows

You are even possible to get query result as `*sql.Row` or `*sql.Rows`

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

// Raw SQL
rows, err := db.Raw("select name, age, email from users where name = ?", "jinzhu").Rows() // (*sql.Rows, error)
defer rows.Close()
for rows.Next() {
	...
	rows.Scan(&name, &age, &email)
	...
}
```

## Scan

Scan results into another struct.

```go
type Result struct {
	Name string
	Age  int
}

var result Result
db.Table("users").Select("name, age").Where("name = ?", 3).Scan(&result)

// Raw SQL
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

## Transactions

All individual save and delete operations are run in a transaction by default.

```go
// begin
tx := db.Begin()

// rollback
tx.Rollback()

// commit
tx.Commit()
```

## Scopes

```go
func AmountGreaterThan1000(db *gorm.DB) *gorm.DB {
	return db.Where("amount > ?", 1000)
}

func PaidWithCreditCard(db *gorm.DB) *gorm.DB {
	return db.Where("pay_mode_sign = ?", "C")
}

func PaidWithCod(db *gorm.DB) *gorm.DB {
	return db.Where("pay_mode_sign = ?", "C")
}

func OrderStatus(status []string) func (db *gorm.DB) *gorm.DB {
	return func (db *gorm.DB) *gorm.DB {
		return db.Scopes(AmountGreaterThan1000).Where("status in (?)", status)
	}
}

db.Scopes(AmountGreaterThan1000, PaidWithCreditCard).Find(&orders)
// Find all credit card orders and amount greater than 1000

db.Scopes(AmountGreaterThan1000, PaidWithCod).Find(&orders)
// Find all COD orders and amount greater than 1000

db.Scopes(OrderStatus([]string{"paid", "shipped"})).Find(&orders)
// Find all paid, shipped orders
```

## Callbacks

Callbacks are methods defined on the pointer of struct.
If any callback returns an error, gorm will stop future operations and rollback all changes.

Here is the list of all available callbacks:
(listed in the same order in which they will get called during the respective operations)

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
// load data from database
AfterFind
```

### Example

```go
func (u *User) BeforeUpdate() (err error) {
	if u.readonly() {
		err = errors.New("read only user")
	}
	return
}

// Rollback the insertion if user's id greater than 1000
func (u *User) AfterCreate() (err error) {
	if (u.Id > 1000) {
		err = errors.New("user id is already greater than 1000")
	}
	return
}
```

As you know, save/delete operations in gorm are running in a transaction,
This is means if changes made in the transaction is not visiable unless it is commited,
So if you want to use those changes in your callbacks, you need to run SQL in same transaction.
Fortunately, gorm support pass transaction to callbacks as you needed, you could do it like this:

```go
func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	tx.Model(u).Update("role", "admin")
	return
}
```

## Specifying The Table Name

```go
// Create `deleted_users` table with struct User's definition
db.Table("deleted_users").CreateTable(&User{})

var deleted_users []User
db.Table("deleted_users").Find(&deleted_users)
//// SELECT * FROM deleted_users;

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

## Error Handling

```go
query := db.Where("name = ?", "jinzhu").First(&user)
query := db.First(&user).Limit(10).Find(&users)
// query.Error will return the last happened error

// So you could do error handing in your application like this:
if err := db.Where("name = ?", "jinzhu").First(&user).Error; err != nil {
	// error handling...
}

// RecordNotFound
// If no record found when you query data, gorm will return RecordNotFound error, you could check it like this:
db.Where("name = ?", "hello world").First(&User{}).Error == gorm.RecordNotFound
// Or use the shortcut method
db.Where("name = ?", "hello world").First(&user).RecordNotFound()

if db.Model(&user).Related(&credit_card).RecordNotFound() {
	// no credit card found error handling
}
```

## Logger

Gorm has built-in logger support

```go
// Enable Logger
db.LogMode(true)

// Diable Logger
db.LogMode(false)

// Debug a single operation
db.Debug().Where("name = ?", "jinzhu").First(&User{})
```

![logger](https://raw.github.com/jinzhu/gorm/master/images/logger.png)

### Customize Logger

```go
// Refer gorm's default logger for how to: https://github.com/jinzhu/gorm/blob/master/logger.go#files
db.SetLogger(gorm.Logger{revel.TRACE})
db.SetLogger(log.New(os.Stdout, "\r\n", 0))
```

## Existing Schema

If you have an existing database schema, and the primary key field is different from `id`, you can add a tag to the field structure to specify that this field is a primary key.

```go
type Animal struct {
	AnimalId     int64 `gorm:"primary_key:yes"`
	Birthday     time.Time
	Age          int64
}
```

If your column names differ from the struct fields, you can specify them like this:

```go
type Animal struct {
	AnimalId    int64     `gorm:"column:beast_id; primary_key:yes"`
	Birthday    time.Time `gorm:"column:day_of_the_beast"`
	Age         int64     `gorm:"column:age_of_the_beast"`
}
```

## More examples with query chain

```go
db.First(&first_article).Count(&total_count).Limit(10).Find(&first_page_articles).Offset(10).Find(&second_page_articles)
//// SELECT * FROM articles LIMIT 1; (first_article)
//// SELECT count(*) FROM articles; (total_count)
//// SELECT * FROM articles LIMIT 10; (first_page_articles)
//// SELECT * FROM articles LIMIT 10 OFFSET 10; (second_page_articles)


db.Where("created_at > ?", "2013-10-10").Find(&cancelled_orders, "state = ?", "cancelled").Find(&shipped_orders, "state = ?", "shipped")
//// SELECT * FROM orders WHERE created_at > '2013/10/10' AND state = 'cancelled'; (cancelled_orders)
//// SELECT * FROM orders WHERE created_at > '2013/10/10' AND state = 'shipped'; (shipped_orders)


// Use variables to keep query chain
todays_orders := db.Where("created_at > ?", "2013-10-29")
cancelled_orders := todays_orders.Where("state = ?", "cancelled")
shipped_orders := todays_orders.Where("state = ?", "shipped")


// Search with shared conditions for different tables
db.Where("product_name = ?", "fancy_product").Find(&orders).Find(&shopping_carts)
//// SELECT * FROM orders WHERE product_name = 'fancy_product'; (orders)
//// SELECT * FROM carts WHERE product_name = 'fancy_product'; (shopping_carts)


// Search with shared conditions from different tables with specified table
db.Where("mail_type = ?", "TEXT").Find(&users1).Table("deleted_users").Find(&users2)
//// SELECT * FROM users WHERE mail_type = 'TEXT'; (users1)
//// SELECT * FROM deleted_users WHERE mail_type = 'TEXT'; (users2)


// FirstOrCreate example
db.Where("email = ?", "x@example.org").Attrs(User{RegisteredIp: "111.111.111.111"}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE email = 'x@example.org';
//// INSERT INTO "users" (email,registered_ip) VALUES ("x@example.org", "111.111.111.111")  // if record not found
```

## TODO
* db.RegisterFuncation("Search", func() {})
  db.Model(&[]User{}).Limit(10).Do("Search", "search func's argument")
  db.Mode(&User{}).Do("EditForm").Get("edit_form_html")
  DefaultValue, DefaultTimeZone, R/W Splitting, Validation
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
