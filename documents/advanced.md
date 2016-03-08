# Advanced Usage

<!-- toc -->

## Error Handling

After perform any operations, if there are any error happened, GORM will set it to `*DB`'s `Error` field

```go
if err := db.Where("name = ?", "jinzhu").First(&user).Error; err != nil {
	// error handling...
}

// If there are more than one error happened, get all of them with `GetErrors`, it returns `[]error`
db.First(&user).Limit(10).Find(&users).GetErrors()

// Check if returns RecordNotFound error
db.Where("name = ?", "hello world").First(&user).RecordNotFound()

if db.Model(&user).Related(&credit_card).RecordNotFound() {
	// no credit card found handling
}
```

## Transactions

To perform a set of operations within a transaction, the general flow is as below.

```go
// begin a transaction
tx := db.Begin()

// do some database operations in the transaction (use 'tx' from this point, not 'db')
tx.Create(...)

// ...

// rollback the transaction in case of error
tx.Rollback()

// Or commit the transaction
tx.Commit()
```

### A Specific Example

```go
func CreateAnimals(db *gorm.DB) err {
  tx := db.Begin()
  // Note the use of tx as the database handle once you are within a transaction

  if err := tx.Create(&Animal{Name: "Giraffe"}).Error; err != nil {
     tx.Rollback()
     return err
  }

  if err := tx.Create(&Animal{Name: "Lion"}).Error; err != nil {
     tx.Rollback()
     return err
  }

  tx.Commit()
  return nil
}
```

## SQL Builder

#### Run Raw SQL

Run Raw SQL

```go
db.Exec("DROP TABLE users;")
db.Exec("UPDATE orders SET shipped_at=? WHERE id IN (?)", time.Now, []int64{11,22,33})

// Scan
type Result struct {
	Name string
	Age  int
}

var result Result
db.Raw("SELECT name, age FROM users WHERE name = ?", 3).Scan(&result)
```

#### sql.Row & sql.Rows

Get query result as `*sql.Row` or `*sql.Rows`

```go
row := db.Table("users").Where("name = ?", "jinzhu").Select("name, age").Row() // (*sql.Row)
row.Scan(&name, &age)

rows, err := db.Model(&User{}).Where("name = ?", "jinzhu").Select("name, age, email").Rows() // (*sql.Rows, error)
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

#### Scan sql.Rows In Iteration

```go
rows, err := db.Model(&User{}).Where("name = ?", "jinzhu").Select("name, age, email").Rows() // (*sql.Rows, error)
defer rows.Close()

for rows.Next() {
  var user User
  db.ScanRows(rows, &user)
  // do something
}
```

## Generic database interface sql.DB

Get generic database interface [*sql.DB](http://golang.org/pkg/database/sql/#DB) from `*gorm.DB` connection

```go
// Get generic database object `*sql.DB` to use its functions
db.DB()

// Ping
db.DB().Ping()
```

#### Connection Pool

```go
db.DB().SetMaxIdleConns(10)
db.DB().SetMaxOpenConns(100)
```

## Composite Primary Key

Set multiple fields as priamry key to enable composite primary key

```go
type Product struct {
	ID           string `gorm:"primary_key"`
	LanguageCode string `gorm:"primary_key"`
}
```

## Logger

Gorm has built-in logger support, by default, it will print happened errors

```go
// Enable Logger, show detailed log
db.LogMode(true)

// Diable Logger, don't show any log
db.LogMode(false)

// Debug a single operation, show detailed log for this operation
db.Debug().Where("name = ?", "jinzhu").First(&User{})
```

#### Customize Logger

Refer GORM's default logger for how to customize it [https://github.com/jinzhu/gorm/blob/master/logger.go](https://github.com/jinzhu/gorm/blob/master/logger.go)

```go
db.SetLogger(gorm.Logger{revel.TRACE})
db.SetLogger(log.New(os.Stdout, "\r\n", 0))
```
