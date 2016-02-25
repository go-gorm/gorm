# Create

### Create Record

```go
user := User{Name: "Jinzhu", Age: 18, Birthday: time.Now()}

db.NewRecord(user) // => returns `true` as primary key is blank

db.Create(&user)

db.NewRecord(user) // => return `false` after `user` created

// Associations will be inserted automatically when save the record
user := User{
	Name:            "jinzhu",
	BillingAddress:  Address{Address1: "Billing Address - Address 1"},
	ShippingAddress: Address{Address1: "Shipping Address - Address 1"},
	Emails:          []Email{{Email: "jinzhu@example.com"}, {Email: "jinzhu-2@example@example.com"}},
	Languages:       []Language{{Name: "ZH"}, {Name: "EN"}},
}

db.Create(&user)
//// BEGIN TRANSACTION;
//// INSERT INTO "addresses" (address1) VALUES ("Billing Address - Address 1");
//// INSERT INTO "addresses" (address1) VALUES ("Shipping Address - Address 1");
//// INSERT INTO "users" (name,billing_address_id,shipping_address_id) VALUES ("jinzhu", 1, 2);
//// INSERT INTO "emails" (user_id,email) VALUES (111, "jinzhu@example.com");
//// INSERT INTO "emails" (user_id,email) VALUES (111, "jinzhu-2@example.com");
//// INSERT INTO "languages" ("name") VALUES ('ZH');
//// INSERT INTO user_languages ("user_id","language_id") VALUES (111, 1);
//// INSERT INTO "languages" ("name") VALUES ('EN');
//// INSERT INTO user_languages ("user_id","language_id") VALUES (111, 2);
//// COMMIT;
```

### Create With Associations

Refer Associations for more details

### Default Values

You could defined default value in the `sql` tag, then the generated creating SQL will ignore these fields that including default value and its value is blank, and after inserted the record into databae, gorm will load those fields's value from database.

```go
type Animal struct {
	ID   int64
	Name string `sql:"default:'galeone'"`
	Age  int64
}

var animal = Animal{Age: 99, Name: ""}
db.Create(&animal)
// INSERT INTO animals("age") values('99');
// SELECT name from animals WHERE ID=111; // the returning primary key is 111
// animal.Name => 'galeone'
```

### Setting Primary Key In Callbacks

If you want to set primary key in `BeforeCreate` callback, you could use `scope.SetColumn`, for example:

```go
func (user *User) BeforeCreate(scope *gorm.Scope) error {
  scope.SetColumn("ID", uuid.New())
  return nil
}
```

### Extra Creating option

```go
// Add extra SQL option for inserting SQL
db.Set("gorm:insert_option", "ON CONFLICT").Create(&product)
// INSERT INTO products (name, code) VALUES ("name", "code") ON CONFLICT;
```
