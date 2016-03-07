# Advanced Usage

<!-- toc -->

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

## Transactions

To perform a set of operations within a transaction, the general flow is as below.
The database handle returned from ``` db.Begin() ``` should be used for all operations within the transaction.
(Note that all individual save and delete operations are run in a transaction by default.)

```go
// begin
tx := db.Begin()

// do some database operations (use 'tx' from this point, not 'db')
tx.Create(...)
...

// rollback in case of error
tx.Rollback()

// Or commit if all is ok
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

## Raw SQL

```go
db.Exec("DROP TABLE users;")
db.Exec("UPDATE orders SET shipped_at=? WHERE id IN (?)", time.Now, []int64{11,22,33})
```

## Row & Rows

It is even possible to get query result as `*sql.Row` or `*sql.Rows`

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

### Scan Rows

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

Get generic database interface from `*gorm.DB` connection [*sql.DB](http://golang.org/pkg/database/sql/#DB)

```go
// Get generic database object *sql.DB to use its functions
db.DB()

// Connection Pool
db.DB().SetMaxIdleConns(10)
db.DB().SetMaxOpenConns(100)

  // Ping
db.DB().Ping()
```

## Composite Primary Key

```go
type Product struct {
	ID           string `gorm:"primary_key"`
	LanguageCode string `gorm:"primary_key"`
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

![logger](https://raw.github.com/jinzhu/gorm/master/doc/logger.png)

### Customize Logger

```go
// Refer gorm's default logger for how to: https://github.com/jinzhu/gorm/blob/master/logger.go#files
db.SetLogger(gorm.Logger{revel.TRACE})
db.SetLogger(log.New(os.Stdout, "\r\n", 0))
```
