# Has Many

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
