package gorm_test

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"reflect"
	"time"
)

type User struct {
	Id                int64
	Age               int64
	UserNum           Num
	Name              string        `sql:"size:255"`
	Birthday          time.Time     // Time
	CreatedAt         time.Time     // CreatedAt: Time of record is created, will be insert automatically
	UpdatedAt         time.Time     // UpdatedAt: Time of record is updated, will be updated automatically
	Emails            []Email       // Embedded structs
	BillingAddress    Address       // Embedded struct
	BillingAddressID  sql.NullInt64 // Embedded struct's foreign key
	ShippingAddress   Address       // Embedded struct
	ShippingAddressId int64         // Embedded struct's foreign key
	CreditCard        CreditCard
	Latitude          float64
	Languages         []Language `gorm:"many2many:user_languages;"`
	CompanyID         int64
	Company           Company
	Role
	PasswordHash      []byte
	IgnoreMe          int64                 `sql:"-"`
	IgnoreStringSlice []string              `sql:"-"`
	Ignored           struct{ Name string } `sql:"-"`
	IgnoredPointer    *User                 `sql:"-"`
}

type CreditCard struct {
	ID        int8
	Number    string
	UserId    sql.NullInt64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

type Email struct {
	Id        int16
	UserId    int
	Email     string `sql:"type:varchar(100);"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Address struct {
	ID        int
	Address1  string
	Address2  string
	Post      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

type Language struct {
	Id    int
	Name  string
	Users []User `gorm:"many2many:user_languages;"`
}

type Product struct {
	Id                    int64
	Code                  string
	Price                 int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
	AfterFindCallTimes    int64
	BeforeCreateCallTimes int64
	AfterCreateCallTimes  int64
	BeforeUpdateCallTimes int64
	AfterUpdateCallTimes  int64
	BeforeSaveCallTimes   int64
	AfterSaveCallTimes    int64
	BeforeDeleteCallTimes int64
	AfterDeleteCallTimes  int64
}

type Company struct {
	Id    int64
	Name  string
	Owner *User `sql:"-"`
}

type Role struct {
	Name string
}

func (role *Role) Scan(value interface{}) error {
	if b, ok := value.([]uint8); ok {
		role.Name = string(b)
	} else {
		role.Name = value.(string)
	}
	return nil
}

func (role Role) Value() (driver.Value, error) {
	return role.Name, nil
}

func (role Role) IsAdmin() bool {
	return role.Name == "admin"
}

type Num int64

func (i *Num) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
	case int64:
		*i = Num(s)
	default:
		return errors.New("Cannot scan NamedInt from " + reflect.ValueOf(src).String())
	}
	return nil
}

type Animal struct {
	Counter    uint64    `gorm:"primary_key:yes"`
	Name       string    `sql:"DEFAULT:'galeone'"`
	From       string    //test reserved sql keyword as field name
	Age        time.Time `sql:"DEFAULT:current_timestamp"`
	unexported string    // unexported value
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type JoinTable struct {
	From uint64
	To   uint64
	Time time.Time `sql:"default: null"`
}

type Post struct {
	Id             int64
	CategoryId     sql.NullInt64
	MainCategoryId int64
	Title          string
	Body           string
	Comments       []*Comment
	Category       Category
	MainCategory   Category
}

type Category struct {
	Id   int64
	Name string
}

type Comment struct {
	Id      int64
	PostId  int64
	Content string
	Post    Post
}

// Scanner
type NullValue struct {
	Id      int64
	Name    sql.NullString `sql:"not null"`
	Age     sql.NullInt64
	Male    sql.NullBool
	Height  sql.NullFloat64
	AddedAt NullTime
}

type NullTime struct {
	Time  time.Time
	Valid bool
}

func (nt *NullTime) Scan(value interface{}) error {
	if value == nil {
		nt.Valid = false
		return nil
	}
	nt.Time, nt.Valid = value.(time.Time), true
	return nil
}

func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

func getPreparedUser(name string, role string) *User {
	var company Company
	DB.Where(Company{Name: role}).FirstOrCreate(&company)

	return &User{
		Name:            name,
		Age:             20,
		Role:            Role{role},
		BillingAddress:  Address{Address1: fmt.Sprintf("Billing Address %v", name)},
		ShippingAddress: Address{Address1: fmt.Sprintf("Shipping Address %v", name)},
		CreditCard:      CreditCard{Number: fmt.Sprintf("123456%v", name)},
		Emails: []Email{
			{Email: fmt.Sprintf("user_%v@example1.com", name)}, {Email: fmt.Sprintf("user_%v@example2.com", name)},
		},
		Company: company,
		Languages: []Language{
			{Name: fmt.Sprintf("lang_1_%v", name)},
			{Name: fmt.Sprintf("lang_2_%v", name)},
		},
	}
}
