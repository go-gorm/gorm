# Models

<!-- toc -->

## Model Defination

```go
type User struct {
	ID           int
	Birthday     time.Time
	Age          int
	Name         string  `sql:"size:255"` // Default size for string is 255, you could reset it with this tag
	Num          int     `sql:"AUTO_INCREMENT"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	IgnoreMe     int `sql:"-"`   // Ignore this field
}
```

## Conventions & Overriding Conventions

### Table name is the pluralized version of struct name

```go
type User struct {} // default table name is `users`

// set User's table name to be `profiles
type (User) TableName() string {
  return "profiles"
}

// Or disable table name's pluralization globally
db.SingularTable(true) // if set this to true, then default table name will be `user`, table name setted with `TableName` won't be affected
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
	AnimalId    int64     `gorm:"column:beast_id"` // set column name to be `beast_id`
	Birthday    time.Time `gorm:"column:day_of_the_beast"` // set column name to be `day_of_the_beast`
	Age         int64     `gorm:"column:age_of_the_beast"` // set column name to be `age_of_the_beast`
}
```

### Default use field `ID` as primary key

```go
type User struct {
  ID uint  // field `ID` is the default primary key for `User`
  Name string
}

type Animal struct {
  // tag `primary_key` used to set `AnimalId` to be primary key
  AnimalId int64 `gorm:"primary_key"`
  Name     string
  Age      int64
}
```

### Use `CreatedAt` to store record's created time if field exists

```go
db.Create(&user) // will set field `CreatedAt`'s time to time now

// If you want to change its value, use `Update`
db.Model(&user).Update("CreatedAt", time.Now())
```

### Use `UpdatedAt` to store record's updated time if field exists
### Use `DeletedAt` to store record's deleted time if field exists
### Gorm provide a default model struct, you could embed it in your struct
