# Associations

<!-- toc -->

## Belongs To

```go
// `User` belongs to `Profile`, `ProfileID` is the foreign key
type User struct {
  gorm.Model
  Profile   Profile
  ProfileID int
}

type Profile struct {
  gorm.Model
  Name string
}

db.Model(&user).Related(&profile)
//// SELECT * FROM profiles WHERE id = 111; // 111 is user's foreign key ProfileID
```

*Specify Foreign Key*

```go
type Profile struct {
	gorm.Model
	Name string
}

type User struct {
	gorm.Model
	Profile      Profile `gorm:"foreignkey:ProfileRefer"` // use ProfileRefer as foreign key
	ProfileRefer uint
}
```

*Specify Foreign Key & Association Key*

```go
type Profile struct {
	gorm.Model
	Refer int
	Name  string
}

type User struct {
	gorm.Model
	Profile   Profile `gorm:"foreignkey:ProfileID;association_foreignkey:Refer"`
	ProfileID int
}
```

## Has One

```go
// User has one CreditCard, UserID is the foreign key
type User struct {
	gorm.Model
	CreditCard   CreditCard
}

type CreditCard struct {
	gorm.Model
	UserID   uint
	Number   string
}

var card CreditCard
db.Model(&user).Related(&card, "CreditCard")
//// SELECT * FROM credit_cards WHERE user_id = 123; // 123 is user's primary key
// CreditCard is user's field name, it means get user's CreditCard relations and fill it into variable card
// If the field name is same as the variable's type name, like above example, it could be omitted, like:
db.Model(&user).Related(&card)
```

*Specify Foreign Key*

```go
type Profile struct {
  gorm.Model
  Name      string
  UserRefer uint
}

type User struct {
  gorm.Model
  Profile Profile `gorm:"foreignkey:UserRefer"`
}
```

*Specify Foreign Key & Association Key*

```go
type Profile struct {
  gorm.Model
  Name   string
  UserID uint
}

type User struct {
  gorm.Model
  Refer   uint
  Profile Profile `gorm:"foreignkey:UserID;association_foreignkey:Refer"`
}
```

## Has Many

```go
// User has many emails, UserID is the foreign key
type User struct {
	gorm.Model
	Emails   []Email
}

type Email struct {
	gorm.Model
	Email   string
	UserID  uint
}

db.Model(&user).Related(&emails)
//// SELECT * FROM emails WHERE user_id = 111; // 111 is user's primary key
```

*Specify Foreign Key*

```go
type Profile struct {
  gorm.Model
  Name      string
  UserRefer uint
}

type User struct {
  gorm.Model
  Profiles []Profile `gorm:"foreignkey:UserRefer"`
}
```

*Specify Foreign Key & Association Key*

```go
type Profile struct {
  gorm.Model
  Name   string
  UserID uint
}

type User struct {
  gorm.Model
  Refer   uint
  Profiles []Profile `gorm:"foreignkey:UserID;association_foreignkey:Refer"`
}
```

## Many To Many

```go
// User has and belongs to many languages, use `user_languages` as join table
type User struct {
	gorm.Model
	Languages         []Language `gorm:"many2many:user_languages;"`
}

type Language struct {
	gorm.Model
	Name string
}

db.Model(&user).Related(&languages, "Languages")
//// SELECT * FROM "languages" INNER JOIN "user_languages" ON "user_languages"."language_id" = "languages"."id" WHERE "user_languages"."user_id" = 111

db.Preload("Languages").First(&user)
```

*With back-reference :*

```go
// User has and belongs to many languages, use `user_languages` as join table
// Make sure the two models are in different files
type User struct {
	gorm.Model
	Languages         []Language `gorm:"many2many:user_languages;"`
}

type Language struct {
	gorm.Model
	Name string
	Users         	  []User     `gorm:"many2many:user_languages;"`
}

db.Model(&language).Related(&users)
//// SELECT * FROM "users" INNER JOIN "user_languages" ON "user_languages"."user_id" = "users"."id" WHERE  ("user_languages"."language_id" IN ('111'))
```

*Specify Foreign Key & Association Key*

```go
type CustomizePerson struct {
  IdPerson string             `gorm:"primary_key:true"`
  Accounts []CustomizeAccount `gorm:"many2many:PersonAccount;association_foreignkey:idAccount;foreignkey:idPerson"`
}

type CustomizeAccount struct {
  IdAccount string `gorm:"primary_key:true"`
  Name      string
}
```

It will create a many2many relationship for those two structs, and their relations will be saved into join table `PersonAccount` with foreign keys `customize_person_id_person` AND `customize_account_id_account`

*Specify jointable's foreign key*

If you want to change join table's foreign keys, you could use tag `association_jointable_foreignkey`, `jointable_foreignkey`

```go
type CustomizePerson struct {
  IdPerson string             `gorm:"primary_key:true"`
  Accounts []CustomizeAccount `gorm:"many2many:PersonAccount;foreignkey:idPerson;association_foreignkey:idAccount;association_jointable_foreignkey:account_id;jointable_foreignkey:person_id;"`
}

type CustomizeAccount struct {
  IdAccount string `gorm:"primary_key:true"`
  Name      string
}
```

### Self-Referencing Many To Many Relationship

To define a self-referencing many2many relationship, you have to change association's foreign key in the join table.

to make it different with source's foreign key, which is generated using struct's name and its primary key, for example:

```go
type User struct {
  gorm.Model
  Friends []*User `gorm:"many2many:friendships;association_jointable_foreignkey:friend_id"`
}
```

GORM will create a join table with foreign key `user_id` and `friend_id`, and use it to save user's self-reference relationship.

Then you can operate it like normal relations, e.g:

```go
DB.Preload("Friends").First(&user, "id = ?", 1)

DB.Model(&user).Association("Friends").Append(&User{Name: "friend1"}, &User{Name: "friend2"})

DB.Model(&user).Association("Friends").Delete(&User{Name: "friend2"})

DB.Model(&user).Association("Friends").Replace(&User{Name: "new friend"})

DB.Model(&user).Association("Friends").Clear()

DB.Model(&user).Association("Friends").Count()
```

## Polymorphism

Supports polymorphic has-many and has-one associations.

```go
  type Cat struct {
    Id    int
    Name  string
    Toy   Toy `gorm:"polymorphic:Owner;"`
  }

  type Dog struct {
    Id   int
    Name string
    Toy  Toy `gorm:"polymorphic:Owner;"`
  }

  type Toy struct {
    Id        int
    Name      string
    OwnerId   int
    OwnerType string
  }
```

Note: polymorphic belongs-to and many-to-many are explicitly NOT supported, and will throw errors.

## Association Mode

Association Mode contains some helper methods to handle relationship things easily.

```go
// Start Association Mode
var user User
db.Model(&user).Association("Languages")
// `user` is the source, it need to be a valid record (contains primary key)
// `Languages` is source's field name for a relationship.
// If those conditions not matched, will return an error, check it with:
// db.Model(&user).Association("Languages").Error


// Query - Find out all related associations
db.Model(&user).Association("Languages").Find(&languages)


// Append - Append new associations for many2many, has_many, will replace current association for has_one, belongs_to
db.Model(&user).Association("Languages").Append([]Language{languageZH, languageEN})
db.Model(&user).Association("Languages").Append(Language{Name: "DE"})


// Delete - Remove relationship between source & passed arguments, won't delete those arguments
db.Model(&user).Association("Languages").Delete([]Language{languageZH, languageEN})
db.Model(&user).Association("Languages").Delete(languageZH, languageEN)


// Replace - Replace current associations with new one
db.Model(&user).Association("Languages").Replace([]Language{languageZH, languageEN})
db.Model(&user).Association("Languages").Replace(Language{Name: "DE"}, languageEN)


// Count - Return the count of current associations
db.Model(&user).Association("Languages").Count()


// Clear - Remove relationship between source & current associations, won't delete those associations
db.Model(&user).Association("Languages").Clear()
```
