package gorm_test

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

type User struct {
	Id                int64
	Age               int64
	UserNum           Num
	Name              string `sql:"size:255"`
	Email             string
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
	CompanyID         *int
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
	CreatedAt time.Time `sql:"not null"`
	UpdatedAt time.Time
	DeletedAt *time.Time
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
	DeletedAt *time.Time
}

type Language struct {
	gorm.Model
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
	Counter    uint64    `gorm:"primary_key:yes;AUTO_INCREMENT"`
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
	gorm.Model
	Name string
}

type Comment struct {
	gorm.Model
	PostId  int64
	Content string
	Post    Post
}

// Scanner
type NullValue struct {
	Id      int64
	Name    sql.NullString  `sql:"not null"`
	Gender  *sql.NullString `sql:"not null"`
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

func runMigration() {
	if err := DB.DropTableIfExists(&User{}).Error; err != nil {
		fmt.Printf("Got error when try to delete table users, %+v\n", err)
	}

	for _, table := range []string{"animals", "user_languages"} {
		DB.Exec(fmt.Sprintf("drop table %v;", table))
	}

	values := []interface{}{&Product{}, &Email{}, &Address{}, &CreditCard{}, &Company{}, &Role{}, &Language{}, &HNPost{}, &EngadgetPost{}, &Animal{}, &User{}, &JoinTable{}, &Post{}, &Category{}, &Comment{}, &Cat{}, &Dog{}, &Toy{}}
	for _, value := range values {
		DB.DropTable(value)
	}

	if err := DB.AutoMigrate(values...).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}
}

func TestIndexes(t *testing.T) {
	if err := DB.Model(&Email{}).AddIndex("idx_email_email", "email").Error; err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}

	scope := DB.NewScope(&Email{})
	if !scope.Dialect().HasIndex(scope.TableName(), "idx_email_email") {
		t.Errorf("Email should have index idx_email_email")
	}

	if err := DB.Model(&Email{}).RemoveIndex("idx_email_email").Error; err != nil {
		t.Errorf("Got error when tried to remove index: %+v", err)
	}

	if scope.Dialect().HasIndex(scope.TableName(), "idx_email_email") {
		t.Errorf("Email's index idx_email_email should be deleted")
	}

	if err := DB.Model(&Email{}).AddIndex("idx_email_email_and_user_id", "user_id", "email").Error; err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}

	if !scope.Dialect().HasIndex(scope.TableName(), "idx_email_email_and_user_id") {
		t.Errorf("Email should have index idx_email_email_and_user_id")
	}

	if err := DB.Model(&Email{}).RemoveIndex("idx_email_email_and_user_id").Error; err != nil {
		t.Errorf("Got error when tried to remove index: %+v", err)
	}

	if scope.Dialect().HasIndex(scope.TableName(), "idx_email_email_and_user_id") {
		t.Errorf("Email's index idx_email_email_and_user_id should be deleted")
	}

	if err := DB.Model(&Email{}).AddUniqueIndex("idx_email_email_and_user_id", "user_id", "email").Error; err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}

	if !scope.Dialect().HasIndex(scope.TableName(), "idx_email_email_and_user_id") {
		t.Errorf("Email should have index idx_email_email_and_user_id")
	}

	if DB.Save(&User{Name: "unique_indexes", Emails: []Email{{Email: "user1@example.comiii"}, {Email: "user1@example.com"}, {Email: "user1@example.com"}}}).Error == nil {
		t.Errorf("Should get to create duplicate record when having unique index")
	}

	var user = User{Name: "sample_user"}
	DB.Save(&user)
	if DB.Model(&user).Association("Emails").Append(Email{Email: "not-1duplicated@gmail.com"}, Email{Email: "not-duplicated2@gmail.com"}).Error != nil {
		t.Errorf("Should get no error when append two emails for user")
	}

	if DB.Model(&user).Association("Emails").Append(Email{Email: "duplicated@gmail.com"}, Email{Email: "duplicated@gmail.com"}).Error == nil {
		t.Errorf("Should get no duplicated email error when insert duplicated emails for a user")
	}

	if err := DB.Model(&Email{}).RemoveIndex("idx_email_email_and_user_id").Error; err != nil {
		t.Errorf("Got error when tried to remove index: %+v", err)
	}

	if scope.Dialect().HasIndex(scope.TableName(), "idx_email_email_and_user_id") {
		t.Errorf("Email's index idx_email_email_and_user_id should be deleted")
	}

	if DB.Save(&User{Name: "unique_indexes", Emails: []Email{{Email: "user1@example.com"}, {Email: "user1@example.com"}}}).Error != nil {
		t.Errorf("Should be able to create duplicated emails after remove unique index")
	}
}

type BigEmail struct {
	Id           int64
	UserId       int64
	Email        string    `sql:"index:idx_email_agent"`
	UserAgent    string    `sql:"index:idx_email_agent"`
	RegisteredAt time.Time `sql:"unique_index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (b BigEmail) TableName() string {
	return "emails"
}

func TestAutoMigration(t *testing.T) {
	DB.AutoMigrate(&Address{})
	if err := DB.Table("emails").AutoMigrate(&BigEmail{}).Error; err != nil {
		t.Errorf("Auto Migrate should not raise any error")
	}

	DB.Save(&BigEmail{Email: "jinzhu@example.org", UserAgent: "pc", RegisteredAt: time.Now()})

	scope := DB.NewScope(&BigEmail{})
	if !scope.Dialect().HasIndex(scope.TableName(), "idx_email_agent") {
		t.Errorf("Failed to create index")
	}

	if !scope.Dialect().HasIndex(scope.TableName(), "uix_emails_registered_at") {
		t.Errorf("Failed to create index")
	}

	var bigemail BigEmail
	DB.First(&bigemail, "user_agent = ?", "pc")
	if bigemail.Email != "jinzhu@example.org" || bigemail.UserAgent != "pc" || bigemail.RegisteredAt.IsZero() {
		t.Error("Big Emails should be saved and fetched correctly")
	}
}
