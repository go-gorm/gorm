package gorm_test

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
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
	Birthday          *time.Time    // Time
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
	Role              Role
	Password          EncryptedData
	PasswordHash      []byte
	IgnoreMe          int64                 `sql:"-"`
	IgnoreStringSlice []string              `sql:"-"`
	Ignored           struct{ Name string } `sql:"-"`
	IgnoredPointer    *User                 `sql:"-"`
}

type NotSoLongTableName struct {
	Id                int64
	ReallyLongThingID int64
	ReallyLongThing   ReallyLongTableNameToTestMySQLNameLengthLimit
}

type ReallyLongTableNameToTestMySQLNameLengthLimit struct {
	Id int64
}

type ReallyLongThingThatReferencesShort struct {
	Id      int64
	ShortID int64
	Short   Short
}

type Short struct {
	Id int64
}

type CreditCard struct {
	ID        int8
	Number    string
	UserId    sql.NullInt64
	CreatedAt time.Time `sql:"not null"`
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"column:deleted_time"`
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

type Place struct {
	Id             int64
	PlaceAddressID int
	PlaceAddress   *Address `gorm:"save_associations:false"`
	OwnerAddressID int
	OwnerAddress   *Address `gorm:"save_associations:true"`
}

type EncryptedData []byte

func (data *EncryptedData) Scan(value interface{}) error {
	if b, ok := value.([]byte); ok {
		if len(b) < 3 || b[0] != '*' || b[1] != '*' || b[2] != '*' {
			return errors.New("Too short")
		}

		*data = b[3:]
		return nil
	}

	return errors.New("Bytes expected")
}

func (data EncryptedData) Value() (driver.Value, error) {
	if len(data) > 0 && data[0] == 'x' {
		//needed to test failures
		return nil, errors.New("Should not start with 'x'")
	}

	//prepend asterisks
	return append([]byte("***"), data...), nil
}

type Role struct {
	Name string `gorm:"size:256"`
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
		n, _ := strconv.Atoi(string(s))
		*i = Num(n)
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
	gorm.Model
	Name string

	Categories []Category
	CategoryID *uint
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

	values := []interface{}{&Short{}, &ReallyLongThingThatReferencesShort{}, &ReallyLongTableNameToTestMySQLNameLengthLimit{}, &NotSoLongTableName{}, &Product{}, &Email{}, &Address{}, &CreditCard{}, &Company{}, &Role{}, &Language{}, &HNPost{}, &EngadgetPost{}, &Animal{}, &User{}, &JoinTable{}, &Post{}, &Category{}, &Comment{}, &Cat{}, &Dog{}, &Hamster{}, &Toy{}, &ElementWithIgnoredField{}, &Place{}}
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

type EmailWithIdx struct {
	Id           int64
	UserId       int64
	Email        string     `sql:"index:idx_email_agent"`
	UserAgent    string     `sql:"index:idx_email_agent"`
	RegisteredAt *time.Time `sql:"unique_index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func TestAutoMigration(t *testing.T) {
	DB.AutoMigrate(&Address{})
	DB.DropTable(&EmailWithIdx{})
	if err := DB.AutoMigrate(&EmailWithIdx{}).Error; err != nil {
		t.Errorf("Auto Migrate should not raise any error")
	}

	now := time.Now()
	DB.Save(&EmailWithIdx{Email: "jinzhu@example.org", UserAgent: "pc", RegisteredAt: &now})

	scope := DB.NewScope(&EmailWithIdx{})
	if !scope.Dialect().HasIndex(scope.TableName(), "idx_email_agent") {
		t.Errorf("Failed to create index")
	}

	if !scope.Dialect().HasIndex(scope.TableName(), "uix_email_with_idxes_registered_at") {
		t.Errorf("Failed to create index")
	}

	var bigemail EmailWithIdx
	DB.First(&bigemail, "user_agent = ?", "pc")
	if bigemail.Email != "jinzhu@example.org" || bigemail.UserAgent != "pc" || bigemail.RegisteredAt.IsZero() {
		t.Error("Big Emails should be saved and fetched correctly")
	}
}

func TestCreateAndAutomigrateTransaction(t *testing.T) {
	tx := DB.Begin()

	func() {
		type Bar struct {
			ID uint
		}
		DB.DropTableIfExists(&Bar{})

		if ok := DB.HasTable("bars"); ok {
			t.Errorf("Table should not exist, but does")
		}

		if ok := tx.HasTable("bars"); ok {
			t.Errorf("Table should not exist, but does")
		}
	}()

	func() {
		type Bar struct {
			Name string
		}
		err := tx.CreateTable(&Bar{}).Error

		if err != nil {
			t.Errorf("Should have been able to create the table, but couldn't: %s", err)
		}

		if ok := tx.HasTable(&Bar{}); !ok {
			t.Errorf("The transaction should be able to see the table")
		}
	}()

	func() {
		type Bar struct {
			Stuff string
		}

		err := tx.AutoMigrate(&Bar{}).Error
		if err != nil {
			t.Errorf("Should have been able to alter the table, but couldn't")
		}
	}()

	tx.Rollback()
}

type MultipleIndexes struct {
	ID     int64
	UserID int64  `sql:"unique_index:uix_multipleindexes_user_name,uix_multipleindexes_user_email;index:idx_multipleindexes_user_other"`
	Name   string `sql:"unique_index:uix_multipleindexes_user_name"`
	Email  string `sql:"unique_index:,uix_multipleindexes_user_email"`
	Other  string `sql:"index:,idx_multipleindexes_user_other"`
}

func TestMultipleIndexes(t *testing.T) {
	if err := DB.DropTableIfExists(&MultipleIndexes{}).Error; err != nil {
		fmt.Printf("Got error when try to delete table multiple_indexes, %+v\n", err)
	}

	DB.AutoMigrate(&MultipleIndexes{})
	if err := DB.AutoMigrate(&EmailWithIdx{}).Error; err != nil {
		t.Errorf("Auto Migrate should not raise any error")
	}

	DB.Save(&MultipleIndexes{UserID: 1, Name: "jinzhu", Email: "jinzhu@example.org", Other: "foo"})

	scope := DB.NewScope(&MultipleIndexes{})
	if !scope.Dialect().HasIndex(scope.TableName(), "uix_multipleindexes_user_name") {
		t.Errorf("Failed to create index")
	}

	if !scope.Dialect().HasIndex(scope.TableName(), "uix_multipleindexes_user_email") {
		t.Errorf("Failed to create index")
	}

	if !scope.Dialect().HasIndex(scope.TableName(), "uix_multiple_indexes_email") {
		t.Errorf("Failed to create index")
	}

	if !scope.Dialect().HasIndex(scope.TableName(), "idx_multipleindexes_user_other") {
		t.Errorf("Failed to create index")
	}

	if !scope.Dialect().HasIndex(scope.TableName(), "idx_multiple_indexes_other") {
		t.Errorf("Failed to create index")
	}

	var mutipleIndexes MultipleIndexes
	DB.First(&mutipleIndexes, "name = ?", "jinzhu")
	if mutipleIndexes.Email != "jinzhu@example.org" || mutipleIndexes.Name != "jinzhu" {
		t.Error("MutipleIndexes should be saved and fetched correctly")
	}

	// Check unique constraints
	if err := DB.Save(&MultipleIndexes{UserID: 1, Name: "name1", Email: "jinzhu@example.org", Other: "foo"}).Error; err == nil {
		t.Error("MultipleIndexes unique index failed")
	}

	if err := DB.Save(&MultipleIndexes{UserID: 1, Name: "name1", Email: "foo@example.org", Other: "foo"}).Error; err != nil {
		t.Error("MultipleIndexes unique index failed")
	}

	if err := DB.Save(&MultipleIndexes{UserID: 2, Name: "name1", Email: "jinzhu@example.org", Other: "foo"}).Error; err == nil {
		t.Error("MultipleIndexes unique index failed")
	}

	if err := DB.Save(&MultipleIndexes{UserID: 2, Name: "name1", Email: "foo2@example.org", Other: "foo"}).Error; err != nil {
		t.Error("MultipleIndexes unique index failed")
	}
}

func TestModifyColumnType(t *testing.T) {
	if dialect := os.Getenv("GORM_DIALECT"); dialect != "postgres" && dialect != "mysql" && dialect != "mssql" {
		t.Skip("Skipping this because only postgres, mysql and mssql support altering a column type")
	}

	type ModifyColumnType struct {
		gorm.Model
		Name1 string `gorm:"length:100"`
		Name2 string `gorm:"length:200"`
	}
	DB.DropTable(&ModifyColumnType{})
	DB.CreateTable(&ModifyColumnType{})

	name2Field, _ := DB.NewScope(&ModifyColumnType{}).FieldByName("Name2")
	name2Type := DB.Dialect().DataTypeOf(name2Field.StructField)

	if err := DB.Model(&ModifyColumnType{}).ModifyColumn("name1", name2Type).Error; err != nil {
		t.Errorf("No error should happen when ModifyColumn, but got %v", err)
	}
}
