# Has One

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
