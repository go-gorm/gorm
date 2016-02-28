# Models

<!-- toc -->

## Model Defination

```go
type User struct {
  gorm.Model
	Birthday     time.Time
	Age          int
	Name         string  `sql:"size:255"` // Default size for string is 255, you could reset it with this tag
	Num          int     `sql:"AUTO_INCREMENT"`
	IgnoreMe     int `sql:"-"`   // Ignore this field
}
```

## Conventions & Overriding Conventions

### `gorm.Model` struct

Gorm has defined struct `gorm.Model`, which could be embeded in your models, it will add fields `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt` to your model

```go
// Model's definition
type Model struct {
  ID        uint `gorm:"primary_key"`
  CreatedAt time.Time
  UpdatedAt time.Time
  DeletedAt *time.Time
}
```

### Table name is the pluralized version of struct name

```go
type User struct {} // default table name is `users`

// set User's table name to be `profiles
type (User) TableName() string {
  return "profiles"
}

func (u User) TableName() string {
	if u.Role == "admin" {
		return "admin_users"
	} else {
		return "users"
	}
}

// Disable table name's pluralization globally
db.SingularTable(true) // if set this to true, `User`'s default table name will be `user`, table name setted with `TableName` won't be affected
```

### Column name is the snake case of field's name

```go
type User struct {
  ID uint             // column name will be `id`
  Name string         // column name will be `name`
  Birthday time.Time  // column name will be `birthday`
  CreatedAt time.Time // column name will be `created_at`
}

type Animal struct {
	AnimalId    int64     `gorm:"column:beast_id"`         // set column name to `beast_id`
	Birthday    time.Time `gorm:"column:day_of_the_beast"` // set column name to `day_of_the_beast`
	Age         int64     `gorm:"column:age_of_the_beast"` // set column name to `age_of_the_beast`
}
```

### Field `ID` as primary key

```go
type User struct {
  ID uint  // field named `ID` is the default primary key for `User`
  Name string
}

// your could also use tag `primary_key` to set other field as primary key
type Animal struct {
  AnimalId int64 `gorm:"primary_key"` // set AnimalId to be primary key
  Name     string
  Age      int64
}
```

### Field `CreatedAt` used to store record's created time

Create records having `CreatedAt` field will set it to current time.

```go
db.Create(&user) // will set `CreatedAt` to current time

// To change its value, you could use `Update`
db.Model(&user).Update("CreatedAt", time.Now())
```

### Use `UpdatedAt` used to store record's updated time

Save records having `UpdatedAt` field will set it to current time.

```go
db.Save(&user) // will set `UpdatedAt` to current time
db.Model(&user).Update("name", "jinzhu") // will set `UpdatedAt` to current time
```

### Use `DeletedAt` to store record's deleted time if field exists

Delete records having `DeletedAt` field, it won't delete the record from database, but will set field `DeletedAt`'s value to current time.
