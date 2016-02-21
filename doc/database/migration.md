## Migration

<!-- toc -->

### Auto Migration

Automatically migrate your schema, to keep your schema update to date

**WARNING** AutoMigrate will ONLY create tables, columns and indexes if doesn't exist,
WON'T change existing column's type or delete unused columns to protect your data

```go
db.AutoMigrate(&User{})

db.AutoMigrate(&User{}, &Product{}, &Order{})

// Add table suffix when create tables
db.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(&User{})
```

### Has Table

```go
// Check if model `User`'s table has been created or not
db.HasTable(&User{})

// Check table `users` exists or not
db.HasTable("users")
```

### Create Table

```go
db.CreateTable(&User{})

db.Set("gorm:table_options", "ENGINE=InnoDB").CreateTable(&User{})
// will append "ENGINE=InnoDB" to the SQL statement when creating table `users`
```

### Drop table

```go
db.DropTable(&User{})
```

### ModifyColumn

Change column's type

```go
// change column description's data type to `text` for model `User`'s table
db.Model(&User{}).ModifyColumn("description", "text")
```

### DropColumn

```go
db.Model(&User{}).DropColumn("description")
```
