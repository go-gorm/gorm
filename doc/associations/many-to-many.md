# Many To Many

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

db.Model(&user).Related(&languages)
//// SELECT * FROM "languages" INNER JOIN "user_languages" ON "user_languages"."language_id" = "languages"."id" WHERE "user_languages"."user_id" = 111
```
