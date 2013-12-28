# GORM

Yet Another ORM library for Go, aims for developer friendly

## Overview

* Chainable API
* Relations
* Callbacks (before/after create/save/update/delete)
* Soft Delete
* Auto Migration
* Transaction
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
  Disable pluralization with `db.SingularTable(true)`, or [specify your table name](#specify-table-name)
* Column name is the snake case of field's name.
* Use `Id int64` field as primary key.
* Use tag `sql` to change field's property, change the tag name with `db.SetTagIdentifier(new_name)`.
* Use `CreatedAt` to store record's created time if it exist.
* Use `UpdatedAt` to store record's updated time if it exist.
* Use `DeletedAt` to store record's deleted time if it exist. [Soft Delete](#soft-delete)

```go
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
    // FYI, "NOT NULL" will only works well with NullXXX Scanner, because golang will initalize a default value for most type...
}
```

## Opening a Database

```go
import "github.com/jinzhu/gorm"
import _ "github.com/lib/pq"
// import _ "github.com/go-sql-driver/mysql"
// import _ "github.com/mattn/go-sqlite3"

db, err := Open("postgres", "user=gorm dbname=gorm sslmode=disable")
// db, err = Open("mysql", "gorm:gorm@/gorm?charset=utf8&parseTime=True")
// db, err = Open("sqlite3", "/tmp/gorm.db")

// Get database connection handle [*sql.DB](http://golang.org/pkg/database/sql/#DB)
d := db.DB()

// With it you could use package `database/sql`'s builtin methods
db.DB().SetMaxIdleConns(10)
db.DB().SetMaxOpenConns(100)
db.DB().Ping()

// By default, table name is plural of struct type, you can use struct type as table name with:
db.SingularTable(true)


// Gorm is goroutines friendly, so you can create a global variable to keep the connection and use it everywhere in your project
// db.go
package db

var DB gorm.DB
func init() {
    var err error
    DB, err = gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
    if err != nil {
        panic(fmt.Sprintf("Got error when connect database, the error is '%v'", err))
    }
}

// user.go
package user
import "db"
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

Feel Free to update your struct, AutoMigrate will keep your database update to date.

FYI, AutoMigrate will only add new columns, won't change current column's type or delete unused columns, to make sure your data is safe

If table doesn't exist when AutoMigrate, gorm will run create table automatically.

(only postgres and mysql supported)

```go
db.AutoMigrate(User{})
```

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

Refer [Query With Related](#query-with-related) for how to find associations

```go
user := User{
        Name:            "jinzhu",
        BillingAddress:  Address{Address1: "Billing Address - Address 1"},
        ShippingAddress: Address{Address1: "Shipping Address - Address 1"},
        Emails:          []Email{{Email: "jinzhu@example.com"}, {Email: "jinzhu-2@example@example.com"}},
}

db.Save(&user)
//// INSERT INTO "addresses" (address1) VALUES ("Billing Address - Address 1");
//// INSERT INTO "addresses" (address1) VALUES ("Shipping Address - Address 1");
//// INSERT INTO "users" (name,billing_address_id,shipping_address_id) VALUES ("jinzhu", 1, 2);
//// INSERT INTO "emails" (user_id,email) VALUES (111, "jinzhu@example.com");
//// INSERT INTO "emails" (user_id,email) VALUES (111, "jinzhu-2@example.com");
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

// Get All records
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

// IN for primary Keys
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

// Multiple Conditions
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

### Update an existing struct

```go
user.Name = "jinzhu 2"
user.Age = 100
db.Save(&user)
//// UPDATE users SET name='jinzhu 2', age=100, updated_at = '2013-11-17 21:34:10' WHERE id=111;
```

### Update one attribute with `Update`

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

### Update multiple attributes with `Updates`

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

### Update attributes without callbacks

```go
db.Model(&user).UpdateColumn("name", "hello")
//// UPDATE users SET name='hello' WHERE id = 111;

db.Model(&user).UpdateColumns(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18 WHERE id = 111;
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

If a struct has DeletedAt field, it will get soft delete ability automatically!

For those don't have the filed, will be deleted from database permanently

```go
db.Delete(&user)
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE id = 111;

// Delete with search condiation
db.Where("age = ?", 20).Delete(&User{})
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE age = 20;

// Soft deleted records will be ignored when search
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

(only support search conditions map and struct)

```go
db.FirstOrInit(&user, User{Name: "non_existing"})
//// User{Name: "non_existing"}

db.Where(User{Name: "Jinzhu"}).FirstOrInit(&user)
//// User{Id: 111, Name: "Jinzhu", Age: 20}

db.FirstOrInit(&user, map[string]interface{}{"name": "jinzhu"})
//// User{Id: 111, Name: "Jinzhu", Age: 20}
```

### FirstOrInit With Attrs

Ignore Attrs's arguments when search, but use them to initialize the struct if no record found.

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

Ignore Assign's arguments when search, but use them to fill the struct regardless record found or not

```go
db.Where(User{Name: "non_existing"}).Assign(User{Age: 20}).FirstOrInit(&user)
//// User{Name: "non_existing", Age: 20}

db.Where(User{Name: "Jinzhu"}).Assign(User{Age: 30}).FirstOrInit(&user)
//// User{Id: 111, Name: "Jinzhu", Age: 30}
```

## FirstOrCreate

Try to get the first record, if failed, will initialize the struct with search conditions and insert it to database

```go
db.FirstOrCreate(&user, User{Name: "non_existing"})
//// User{Id: 112, Name: "non_existing"}

db.Where(User{Name: "Jinzhu"}).FirstOrCreate(&user)
//// User{Id: 111, Name: "Jinzhu"}

db.FirstOrCreate(&user, map[string]interface{}{"name": "jinzhu", "age": 30})
//// user -> User{Id: 111, Name: "jinzhu", Age: 20}
```

### FirstOrCreate With Attrs

Ignore Attrs's arguments when search, but use them to initialize the struct if no record found.

```go
db.Where(User{Name: "non_existing"}).Attrs(User{Age: 20}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE name = 'non_existing';
//// User{Id: 112, Name: "non_existing", Age: 20}

db.Where(User{Name: "jinzhu"}).Attrs(User{Age: 30}).FirstOrCreate(&user)
//// User{Id: 111, Name: "jinzhu", Age: 20}
```

### FirstOrCreate With Assign

Ignore Assign's arguments when search, but use them to fill the struct regardless record found or not, then save it back to database

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

// Pluck more than one column? Do it like this
db.Select("name, age").Find(&users)
```

## Callbacks

Callbacks are functions defined to struct's pointer, they would be run when save a struct to database.
If any callback return error, gorm will stop future operations and rollback all changes

Here is a list with all available callbacks,
listed in the same order in which they will get called during the respective operations.

### Creating an Object

```go
BeforeSave
BeforeCreate
// save before associations
// save self
// save after associations
AfterCreate
AfterSave
```
### Updating an Object

```go
BeforeSave
BeforeUpdate
// save before associations
// save self
// save after associations
AfterUpdate
AfterSave
```

### Destroying an Object

```go
BeforeDelete
// delete self
AfterDelete
```

Here is an example:

```go
func (u *User) BeforeUpdate() (err error) {
    if u.readonly() {
        err = errors.New("Read Only User!")
    }
    return
}

// Rollback the insertion if have more than 1000 users
func (u *User) AfterCreate() (err error) {
    if (u.Id > 1000) { // Just as example, don't use Id to count users!
        err = errors.New("Only 1000 users allowed!")
    }
    return
}
```

```go
// As you know, the save/delete operations are running in a transaction
// This is means all your changes will be rollbacked if get any errors
// If you want your changes in callbacks be run in the same transaction
// You have to pass the transaction as argument to the function
func (u *User) AfterCreate(tx *gorm.DB) (err error) {
    tx.Model(u).Update("role", "admin")
    return
}
```

## Specify Table Name

```go
// Create `deleted_users` table with User's fields
db.Table("deleted_users").CreateTable(&User{})

// Search from table `deleted_users`, and fill results to []User
var deleted_users []User
db.Table("deleted_users").Find(&deleted_users)
//// SELECT * FROM deleted_users;

// Delete results from table `deleted_users` with search conditions
db.Table("deleted_users").Where("name = ?", "jinzhu").Delete()
//// DELETE FROM deleted_users WHERE name = 'jinzhu';
```

### Specify Table Name for Struct permanently with TableName method

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

## Transaction

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

Grom has builtin logger support, enable it with:

```go
db.LogMode(true)
```

![logger](https://raw.github.com/jinzhu/gorm/master/images/logger.png)

```go
// Use your own logger
// Refer gorm's default logger for how to format messages: https://github.com/jinzhu/gorm/blob/master/logger.go#files
db.SetLogger(log.New(os.Stdout, "\r\n", 0))

// Disable log
db.LogMode(false)

// Enable log for a single operation, make debug easy
db.Debug().Where("name = ?", "jinzhu").First(&User{})
```

## Row & Rows

Row & Rows is not chainable, it works just like `QueryRow` and `Query`

```go
row := db.Where("name = ?", "jinzhu").select("name, age").Row() // (*sql.Row)
row.Scan(&name, &age)

rows, err := db.Where("name = ?", "jinzhu").select("name, age, email").Rows() // (*sql.Rows, error)
defer rows.Close()
for rows.Next() {
  ...
  rows.Scan(&name, &age, &email)
  ...
}
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
```

## Run Raw SQl

```go
// Raw sql
db.Exec("drop table users;")

// Raw sql with arguments
db.Exec("update orders set shipped_at=? where id in (?)", time.Now, []int64{11,22,33})
```

## Error Handling

```go
query := db.Where("name = ?", "jinzhu").First(&user)
query := db.First(&user).Limit(10).Find(&users)
//// query.Error keep the latest error happened
//// query.Errors keep all errors happened
//// If an error happened, gorm will stop following operations

// I often use some code like below to do error handling when writting applicatoins
if err := db.Where("name = ?", "jinzhu").First(&user).Error; err != nil {
  // ...
}

// If no record found, gorm will return RecordNotFound error, you could check it with
db.Where("name = ?", "hello world").First(&User{}).Error == gorm.RecordNotFound

// Or use shortcut method
if db.Where("name = ?", "hello world").First(&user).RecordNotFound() {
  panic("no record found")
} else {
  user.Blalala()
}

if db.Model(&user).Related(&credit_card).RecordNotFound() {
  panic("no credit card found")
}
```

## Advanced Usage With Query Chain

Already excited about above usage? Let's see some magic!

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


// Use variable to keep query chain
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


// An example for how to use FirstOrCreate
db.Where("email = ?", "x@example.org").Attrs(User{RegisteredIp: "111.111.111.111"}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE email = 'x@example.org';
//// INSERT INTO "users" (email,registered_ip) VALUES ("x@example.org", "111.111.111.111")  // if no record found
```

## TODO
* Joins
* Scan
* AlertColumn, DropColumn, AddIndex, RemoveIndex
* Includes
* Validations

# Author

**jinzhu**

* <http://github.com/jinzhu>
* <wosmvp@gmail.com>
* <http://twitter.com/zhangjinzhu>

## License

Released under the [MIT License](http://www.opensource.org/licenses/MIT).
