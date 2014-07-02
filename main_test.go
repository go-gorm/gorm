package gorm_test

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	testdb "github.com/erikstmartin/go-testdb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"os"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type IgnoredEmbedStruct struct {
	Name string
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

type Company struct {
	Id   int64
	Name string
}

type Role struct {
	Name string
}

func (role *Role) Scan(value interface{}) error {
	role.Name = string(value.([]uint8))
	return nil
}

func (role Role) Value() (driver.Value, error) {
	return role.Name, nil
}

func (role Role) IsAdmin() bool {
	return role.Name == "admin"
}

type User struct {
	Id                 int64 // Id: Primary key
	Age                int64
	UserNum            Num
	Name               string             `sql:"size:255"`
	Birthday           time.Time          // Time
	CreatedAt          time.Time          // CreatedAt: Time of record is created, will be insert automatically
	UpdatedAt          time.Time          // UpdatedAt: Time of record is updated, will be updated automatically
	DeletedAt          time.Time          // DeletedAt: Time of record is deleted, refer Soft Delete for more
	Emails             []Email            // Embedded structs
	IgnoredEmbedStruct IgnoredEmbedStruct `sql:"-"`
	BillingAddress     Address            // Embedded struct
	BillingAddressId   sql.NullInt64      // Embedded struct's foreign key
	ShippingAddress    Address            // Embedded struct
	ShippingAddressId  int64              // Embedded struct's foreign key
	When               time.Time
	CreditCard         CreditCard
	Latitude           float64
	CompanyId          int64
	Company
	Role
	PasswordHash      []byte
	IgnoreMe          int64    `sql:"-"`
	IgnoreStringSlice []string `sql:"-"`
}

type UserCompany struct {
	Id        int64
	UserId    int64
	CompanyId int64
}

func (t UserCompany) TableName() string {
	return "user_companies"
}

type CreditCard struct {
	Id        int8
	Number    string
	UserId    sql.NullInt64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

type Email struct {
	Id        int16
	UserId    int
	Email     string `sql:"type:varchar(100); unique"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Address struct {
	Id        int
	Address1  string
	Address2  string
	Post      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
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

type Animal struct {
	Counter   int64 `primaryKey:"yes"`
	Name      string
	From      string //test reserverd sql keyword as field name
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Details struct {
	Id   int64
	Bulk gorm.Hstore
}

var (
	db                 gorm.DB
	t1, t2, t3, t4, t5 time.Time
)

func init() {
	var err error
	switch os.Getenv("GORM_DIALECT") {
	case "mysql":
		// CREATE USER 'gorm'@'localhost' IDENTIFIED BY 'gorm';
		// CREATE DATABASE gorm;
		// GRANT ALL ON gorm.* TO 'gorm'@'localhost';
		fmt.Println("testing mysql...")
		db, err = gorm.Open("mysql", "gorm:gorm@/gorm?charset=utf8&parseTime=True")
	case "postgres":
		fmt.Println("testing postgres...")
		db, err = gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
	default:
		fmt.Println("testing sqlite3...")
		db, err = gorm.Open("sqlite3", "/tmp/gorm.db")
	}

	// db.SetLogger(Logger{log.New(os.Stdout, "\r\n", 0)})
	// db.SetLogger(log.New(os.Stdout, "\r\n", 0))
	db.LogMode(true)
	db.LogMode(false)

	if err != nil {
		panic(fmt.Sprintf("No error should happen when connect database, but got %+v", err))
	}

	db.DB().SetMaxIdleConns(10)

	if err := db.DropTable(&User{}).Error; err != nil {
		fmt.Printf("Got error when try to delete table users, %+v\n", err)
	}

	db.Exec("drop table products;")
	db.Exec("drop table emails;")
	db.Exec("drop table addresses")
	db.Exec("drop table credit_cards")
	db.Exec("drop table roles")
	db.Exec("drop table companies")
	db.Exec("drop table animals")
	db.Exec("drop table user_companies")

	if err = db.CreateTable(&Animal{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err = db.CreateTable(&User{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err = db.CreateTable(&Product{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err = db.CreateTable(Email{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err = db.AutoMigrate(Address{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err = db.AutoMigrate(&CreditCard{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err = db.AutoMigrate(Company{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err = db.AutoMigrate(Role{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err = db.AutoMigrate(UserCompany{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	var shortForm = "2006-01-02 15:04:05"
	t1, _ = time.Parse(shortForm, "2000-10-27 12:02:40")
	t2, _ = time.Parse(shortForm, "2002-01-01 00:00:00")
	t3, _ = time.Parse(shortForm, "2005-01-01 00:00:00")
	t4, _ = time.Parse(shortForm, "2010-01-01 00:00:00")
	t5, _ = time.Parse(shortForm, "2020-01-01 00:00:00")
	db.Save(&User{Name: "1", Age: 18, Birthday: t1, When: time.Now(), UserNum: Num(111)})
	db.Save(&User{Name: "2", Age: 20, Birthday: t2})
	db.Save(&User{Name: "3", Age: 22, Birthday: t3})
	db.Save(&User{Name: "3", Age: 24, Birthday: t4})
	db.Save(&User{Name: "5", Age: 26, Birthday: t4})

	db.Save(&Animal{Name: "First", From: "hello"})
	db.Save(&Animal{Name: "Amazing", From: "nerdz"})
	db.Save(&Animal{Name: "Horse", From: "gorm"})
	db.Save(&Animal{Name: "Last", From: "epic"})
}

func TestFirstAndLast(t *testing.T) {
	var user1, user2, user3, user4 User
	db.First(&user1)
	db.Order("id").Find(&user2)

	db.Last(&user3)
	db.Order("id desc").Find(&user4)
	if user1.Id != user2.Id || user3.Id != user4.Id {
		t.Errorf("First and Last should work correctly")
	}

	var users []User
	db.First(&users)
	if len(users) != 1 {
		t.Errorf("Find first record as map")
	}
}

func TestFirstAndLastWithJoins(t *testing.T) {
	var user1, user2, user3, user4 User
	db.Joins("left join emails on emails.user_id = users.id").First(&user1)
	db.Order("id").Find(&user2)

	db.Joins("left join emails on emails.user_id = users.id").Last(&user3)
	db.Order("id desc").Find(&user4)
	if user1.Id != user2.Id || user3.Id != user4.Id {
		t.Errorf("First and Last should work correctly with Joins")
	}
}

func TestFirstAndLastForTableWithNoStdPrimaryKey(t *testing.T) {
	var animal1, animal2, animal3, animal4 Animal
	db.First(&animal1)
	db.Order("counter").Find(&animal2)

	db.Last(&animal3)
	db.Order("counter desc").Find(&animal4)
	if animal1.Counter != animal2.Counter || animal3.Counter != animal4.Counter {
		t.Errorf("First and Last should work correctly")
	}

	var animals []Animal
	db.First(&animals)
	if len(animals) != 1 {
		t.Errorf("Find first record as map")
	}
}

func TestSaveCustomType(t *testing.T) {
	var user, user1 User
	db.First(&user, "name = ?", "1")
	if user.UserNum != Num(111) {
		t.Errorf("UserNum should be saved correctly")
	}

	user.UserNum = Num(222)
	db.Save(&user)

	db.First(&user1, "name = ?", "1")
	if user1.UserNum != Num(222) {
		t.Errorf("UserNum should be updated correctly")
	}
}

func TestPrecision(t *testing.T) {
	f := 35.03554004971999
	user := User{Name: "Precision", Latitude: f}
	db.Save(&user)
	if user.Latitude != f {
		t.Errorf("Float64 should not be changed after save")
	}

	var u User
	db.First(&u, "name = ?", "Precision")

	if u.Latitude != f {
		t.Errorf("Float64 should not be changed after query")
	}
}

func TestCreateAndUpdate(t *testing.T) {
	name, name2, new_name := "update", "update2", "new_update"
	user := User{Name: name, Age: 1, PasswordHash: []byte{'f', 'a', 'k', '4'}}

	if !db.NewRecord(user) || !db.NewRecord(&user) {
		t.Error("User should be new record")
	}

	if count := db.Save(&user).RowsAffected; count != 1 {
		t.Error("There should be one record be affected when create record")
	}

	if user.Id == 0 {
		t.Errorf("Should have ID after create")
	}

	if db.NewRecord(user) || db.NewRecord(&user) {
		t.Error("User should not new record after save")
	}

	var u User
	db.First(&u, user.Id)
	if !reflect.DeepEqual(u.PasswordHash, []byte{'f', 'a', 'k', '4'}) {
		t.Errorf("User's Password should be saved")
	}

	if count := db.Save(&User{Name: name2, Age: 1}).RowsAffected; count != 1 {
		t.Error("There should be one record be affected when update a record")
	}

	user.Name = new_name
	db.Save(&user)
	if db.Where("name = ?", name).First(&User{}).Error != gorm.RecordNotFound {
		t.Errorf("Should raise RecordNotFound error when looking with an outdated name")
	}

	if db.Where("name = ?", new_name).First(&User{}).Error != nil {
		t.Errorf("Shouldn't raise error when looking with the new name")
	}

	if db.Where("name = ?", name2).First(&User{}).Error != nil {
		t.Errorf("Shouldn't update other users when update one")
	}
}

func TestDelete(t *testing.T) {
	name, name2 := "delete", "delete2"
	user := User{Name: name, Age: 1}
	db.Save(&user)
	db.Save(&User{Name: name2, Age: 1})

	if db.Delete(&user).Error != nil {
		t.Errorf("Shouldn't raise any error when delete a user")
	}

	if db.Where("name = ?", name).First(&User{}).Error == nil {
		t.Errorf("User can't be found after delete")
	}

	if db.Where("name = ?", name2).First(&User{}).Error != nil {
		t.Errorf("Other users shouldn't be deleted")
	}
}

func TestWhere(t *testing.T) {
	name := "where"
	db.Save(&User{Name: name, Age: 1})

	user := &User{}
	db.Where("name = ?", name).First(user)
	if user.Name != name {
		t.Errorf("Search user with name")
	}

	if db.Where(user.Id).First(&User{}).Error != nil {
		t.Errorf("Search user with primary key")
	}

	if db.Where("name LIKE ?", "%nonono%").First(&User{}).Error == nil {
		t.Errorf("Search non-existing user with regexp name")
	}

	if db.Where("name LIKE ?", "%whe%").First(&User{}).Error != nil {
		t.Errorf("Search user with regexp name")
	}

	if db.Where("name = ?", "non-existing user").First(&User{}).Error == nil {
		t.Errorf("Search non-existing user should get error")
	}

	var users []User
	if !db.Where("name = ?", "none-noexisting").Find(&users).RecordNotFound() {
		t.Errorf("Should get RecordNotFound error when looking for none existing records")
	}
}

func TestComplexWhere(t *testing.T) {
	var users []User
	db.Where("age > ?", 20).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users that age > 20, but got %v", len(users))
	}

	var user_ids []int64
	db.Table("users").Where("age > ?", 20).Pluck("id", &user_ids)
	if len(user_ids) != 3 {
		t.Errorf("Should found 3 users that age > 20, but got %v", len(users))
	}

	users = []User{}
	db.Where(user_ids).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users that age > 20 when search with primary keys, but got %v", len(users))
	}

	users = []User{}
	db.Where("age >= ?", 20).Find(&users)
	if len(users) != 4 {
		t.Errorf("Should found 4 users that age >= 20, but got %v", len(users))
	}

	users = []User{}
	db.Where("age = ?", 20).Find(&users)
	if len(users) != 1 {
		t.Errorf("Should found 1 users age == 20, but got %v", len(users))
	}

	users = []User{}
	db.Where("age <> ?", 20).Find(&users)
	if len(users) < 3 {
		t.Errorf("Should found more than 3 users age != 20, but got %v", len(users))
	}

	users = []User{}
	db.Where("name = ? and age >= ?", "3", 20).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users that age >= 20 with name 3, but got %v", len(users))
	}

	users = []User{}
	db.Where("name = ?", "3").Where("age >= ?", 20).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users that age >= 20 with name 3, but got %v", len(users))
	}

	users = []User{}
	db.Where("birthday > ?", t2).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users's birthday >= t2, but got %v", len(users))
	}

	users = []User{}
	db.Where("birthday > ?", "2002-10-10").Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users's birthday >= 2002-10-10, but got %v", len(users))
	}

	users = []User{}
	db.Where("birthday >= ?", t1).Where("birthday < ?", t2).Find(&users)
	if len(users) != 1 {
		t.Errorf("Should found 1 users's birthday <= t2, but got %v", len(users))
	}

	users = []User{}
	db.Where("birthday >= ? and birthday <= ?", t1, t2).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users's birthday <= t2, but got %v", len(users))
	}

	users = []User{}
	db.Where("name in (?)", []string{"1", "3"}).Find(&users)

	if len(users) != 3 {
		t.Errorf("Should found 3 users's name in (1, 3), but got  %v", len(users))
	}

	user_ids = []int64{}
	for _, user := range users {
		user_ids = append(user_ids, user.Id)
	}

	users = []User{}
	db.Where("id in (?)", user_ids).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users's name in (1, 3), but got %v", len(users))
	}

	users = []User{}
	db.Where("id in (?)", user_ids[0]).Find(&users)
	if len(users) != 1 {
		t.Errorf("Should found 1 users's name in (1), but got %v", len(users))
	}

	users = []User{}
	db.Where("name in (?)", []string{"1", "2"}).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users's name in (1, 2), but got %v", len(users))
	}
}

func TestSearchWithStruct(t *testing.T) {
	var user User
	db.First(&user, &User{Name: "2"})
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Search first record with inline struct pointer")
	}

	db.First(&user, User{Name: "2"})
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Search first record with inline struct")
	}

	db.Where(&User{Name: "2"}).First(&user)
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Search first record with where struct")
	}

	var users []User
	db.Find(&users, &User{Name: "3"})
	if len(users) != 2 {
		t.Errorf("Search all records with inline struct")
	}

	db.Where(User{Name: "3"}).Find(&users)
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Search all records with where struct")
	}
}

func TestSearchWithMap(t *testing.T) {
	var user User
	db.First(&user, map[string]interface{}{"name": "2"})
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Search first record with inline map")
	}

	db.Where(map[string]interface{}{"name": "2"}).First(&user)
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Search first record with where map")
	}

	var users []User
	db.Find(&users, map[string]interface{}{"name": "3"})
	if len(users) != 2 {
		t.Errorf("Search all records with inline map")
	}

	db.Where(map[string]interface{}{"name": "3"}).Find(&users)
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Search all records with where map")
	}
}

func TestInitlineCondition(t *testing.T) {
	var u1, u2, u3, u4, u5, u6, u7 User
	db.Where("name = ?", "3").Order("age desc").First(&u1).First(&u2)

	db.Where("name = ?", "3").First(&u3, "age = 22").First(&u4, "age = ?", 24).First(&u5, "age = ?", 26)
	if !((u5.Id == 0) && (u3.Age == 22 && u3.Name == "3") && (u4.Age == 24 && u4.Name == "3")) {
		t.Errorf("Find first record with inline condition and where")
	}

	db.First(&u6, u1.Id)
	if !(u6.Id == u1.Id && u6.Id != 0) {
		t.Errorf("Find first record with primary key")
	}

	db.First(&u7, strconv.Itoa(int(u1.Id)))
	if !(u6.Id == u1.Id && u6.Id != 0) {
		t.Errorf("Find first record with string primary key")
	}

	var us1, us2, us3, us4 []User
	db.Find(&us1, "age = 22").Find(&us2, "name = ?", "3").Find(&us3, "age > ?", 20)
	if !(len(us1) == 1 && len(us2) == 2 && len(us3) == 3) {
		t.Errorf("Find all records with inline condition and where")
	}

	db.Find(&us4, "name = ? and age > ?", "3", "22")
	if len(us4) != 1 {
		t.Errorf("Find all records with complex inline condition")
	}
}

func TestSelect(t *testing.T) {
	var user User
	db.Where("name = ?", "3").Select("name").Find(&user)
	if user.Id != 0 {
		t.Errorf("Should not have ID because only searching age, %+v", user.Id)
	}

	if user.Name != "3" {
		t.Errorf("Should got Name = 3 when searching with it, %+v", user.Id)
	}
}

func TestOrderAndPluck(t *testing.T) {
	var ages, ages1, ages2, ages3, ages4, ages5 []int64
	db.Model(&[]User{}).Order("age desc").Pluck("age", &ages)
	if ages[0] != 26 {
		t.Errorf("The first age should be 26 when order with age desc")
	}

	db.Model([]User{}).Order("age desc").Pluck("age", &ages1).Order("age").Pluck("age", &ages2)
	if !reflect.DeepEqual(ages1, ages2) {
		t.Errorf("The first order is the primary order")
	}

	db.Model(&User{}).Order("age desc").Pluck("age", &ages3).Order("age", true).Pluck("age", &ages4)
	if reflect.DeepEqual(ages3, ages4) {
		t.Errorf("Reorder should work")
	}

	var names []string
	db.Model(User{}).Order("name").Order("age desc").Pluck("age", &ages5).Pluck("name", &names)
	if !(names[0] == "1" && names[2] == "3" && names[3] == "3" && ages5[2] == 24 && ages5[3] == 22) {
		t.Errorf("Order with multiple orders")
	}

	db.Model(User{}).Select("name, age").Find(&[]User{})
}

func TestLimit(t *testing.T) {
	var users1, users2, users3 []User
	db.Order("age desc").Limit(3).Find(&users1).Limit(5).Find(&users2).Limit(-1).Find(&users3)

	if len(users1) != 3 || len(users2) != 5 || len(users3) <= 5 {
		t.Errorf("Limit should work")
	}
}

func TestOffset(t *testing.T) {
	var users1, users2, users3, users4 []User
	db.Limit(100).Order("age desc").Find(&users1).Offset(3).Find(&users2).Offset(5).Find(&users3).Offset(-1).Find(&users4)

	if (len(users1) != len(users4)) || (len(users1)-len(users2) != 3) || (len(users1)-len(users3) != 5) {
		t.Errorf("Offset should work")
	}
}

func TestOr(t *testing.T) {
	var users []User
	db.Where("name = ?", "1").Or("name = ?", "3").Find(&users)
	if len(users) != 3 {
		t.Errorf("Find users with or")
	}
}

func TestCount(t *testing.T) {
	var count, count1, count2 int64
	var users []User

	if err := db.Where("name = ?", "1").Or("name = ?", "3").Find(&users).Count(&count).Error; err != nil {
		t.Errorf("Count should work", err)
	}

	if count != int64(len(users)) {
		t.Errorf("Count() method should get correct value")
	}

	db.Model(&User{}).Where("name = ?", "1").Count(&count1).Or("name = ?", "3").Count(&count2)
	if count1 != 1 || count2 != 3 {
		t.Errorf("Multiple count should work")
	}
}

func TestCreatedAtAndUpdatedAt(t *testing.T) {
	name := "check_created_at_and_updated_at"
	u := User{Name: name, Age: 1}
	db.Save(&u)
	created_at := u.CreatedAt
	updated_at := u.UpdatedAt

	if created_at.IsZero() {
		t.Errorf("Should have created_at after create")
	}
	if updated_at.IsZero() {
		t.Errorf("Should have updated_at after create")
	}

	u.Name = "check_created_at_and_updated_at_2"
	db.Save(&u)
	created_at2 := u.CreatedAt
	updated_at2 := u.UpdatedAt

	if created_at != created_at2 {
		t.Errorf("Created At should not changed after update")
	}

	if updated_at == updated_at2 {
		t.Errorf("Updated At should be changed after update")
	}
}

func (s *Product) BeforeCreate() (err error) {
	if s.Code == "Invalid" {
		err = errors.New("invalid product")
	}
	s.BeforeCreateCallTimes = s.BeforeCreateCallTimes + 1
	return
}

func (s *Product) BeforeUpdate() (err error) {
	if s.Code == "dont_update" {
		err = errors.New("Can't update")
	}
	s.BeforeUpdateCallTimes = s.BeforeUpdateCallTimes + 1
	return
}

func (s *Product) BeforeSave() (err error) {
	if s.Code == "dont_save" {
		err = errors.New("Can't save")
	}
	s.BeforeSaveCallTimes = s.BeforeSaveCallTimes + 1
	return
}

func (s *Product) AfterFind() {
	s.AfterFindCallTimes = s.AfterFindCallTimes + 1
}

func (s *Product) AfterCreate(db *gorm.DB) {
	db.Model(s).UpdateColumn(Product{AfterCreateCallTimes: s.AfterCreateCallTimes + 1})
}

func (s *Product) AfterUpdate() {
	s.AfterUpdateCallTimes = s.AfterUpdateCallTimes + 1
}

func (s *Product) AfterSave() (err error) {
	if s.Code == "after_save_error" {
		err = errors.New("Can't save")
	}
	s.AfterSaveCallTimes = s.AfterSaveCallTimes + 1
	return
}

func (s *Product) BeforeDelete() (err error) {
	if s.Code == "dont_delete" {
		err = errors.New("Can't delete")
	}
	s.BeforeDeleteCallTimes = s.BeforeDeleteCallTimes + 1
	return
}

func (s *Product) AfterDelete() (err error) {
	if s.Code == "after_delete_error" {
		err = errors.New("Can't delete")
	}
	s.AfterDeleteCallTimes = s.AfterDeleteCallTimes + 1
	return
}

func (p *Product) GetCallTimes() []int64 {
	return []int64{p.BeforeCreateCallTimes, p.BeforeSaveCallTimes, p.BeforeUpdateCallTimes, p.AfterCreateCallTimes, p.AfterSaveCallTimes, p.AfterUpdateCallTimes, p.BeforeDeleteCallTimes, p.AfterDeleteCallTimes, p.AfterFindCallTimes}
}

func TestRunCallbacks(t *testing.T) {
	p := Product{Code: "unique_code", Price: 100}
	db.Save(&p)

	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 1, 0, 1, 1, 0, 0, 0, 0}) {
		t.Errorf("Callbacks should be invoked successfully, %v", p.GetCallTimes())
	}

	db.Where("Code = ?", "unique_code").First(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 1, 0, 1, 0, 0, 0, 0, 1}) {
		t.Errorf("After callbacks values are not saved, %v", p.GetCallTimes())
	}

	p.Price = 200
	db.Save(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 2, 1, 1, 1, 1, 0, 0, 1}) {
		t.Errorf("After update callbacks should be invoked successfully, %v", p.GetCallTimes())
	}

	var products []Product
	db.Find(&products, "code = ?", "unique_code")
	if products[0].AfterFindCallTimes != 2 {
		t.Errorf("AfterFind callbacks should work with slice")
	}

	db.Where("Code = ?", "unique_code").First(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 2, 1, 1, 0, 0, 0, 0, 2}) {
		t.Errorf("After update callbacks values are not saved, %v", p.GetCallTimes())
	}

	db.Delete(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 2, 1, 1, 0, 0, 1, 1, 2}) {
		t.Errorf("After delete callbacks should be invoked successfully, %v", p.GetCallTimes())
	}

	if db.Where("Code = ?", "unique_code").First(&p).Error == nil {
		t.Errorf("Can't find a deleted record")
	}
}

func TestCallbacksWithErrors(t *testing.T) {
	p := Product{Code: "Invalid", Price: 100}
	if db.Save(&p).Error == nil {
		t.Errorf("An error from before create callbacks happened when create with invalid value")
	}

	if db.Where("code = ?", "Invalid").First(&Product{}).Error == nil {
		t.Errorf("Should not save record that have errors")
	}

	if db.Save(&Product{Code: "dont_save", Price: 100}).Error == nil {
		t.Errorf("An error from after create callbacks happened when create with invalid value")
	}

	p2 := Product{Code: "update_callback", Price: 100}
	db.Save(&p2)

	p2.Code = "dont_update"
	if db.Save(&p2).Error == nil {
		t.Errorf("An error from before update callbacks happened when update with invalid value")
	}

	if db.Where("code = ?", "update_callback").First(&Product{}).Error != nil {
		t.Errorf("Record Should not be updated due to errors happened in before update callback")
	}

	if db.Where("code = ?", "dont_update").First(&Product{}).Error == nil {
		t.Errorf("Record Should not be updated due to errors happened in before update callback")
	}

	p2.Code = "dont_save"
	if db.Save(&p2).Error == nil {
		t.Errorf("An error from before save callbacks happened when update with invalid value")
	}

	p3 := Product{Code: "dont_delete", Price: 100}
	db.Save(&p3)
	if db.Delete(&p3).Error == nil {
		t.Errorf("An error from before delete callbacks happened when delete")
	}

	if db.Where("Code = ?", "dont_delete").First(&p3).Error != nil {
		t.Errorf("An error from before delete callbacks happened")
	}

	p4 := Product{Code: "after_save_error", Price: 100}
	db.Save(&p4)
	if err := db.First(&Product{}, "code = ?", "after_save_error").Error; err == nil {
		t.Errorf("Record should be reverted if get an error in after save callback", err)
	}

	p5 := Product{Code: "after_delete_error", Price: 100}
	db.Save(&p5)
	if err := db.First(&Product{}, "code = ?", "after_delete_error").Error; err != nil {
		t.Errorf("Record should be found", err)
	}

	db.Delete(&p5)
	if err := db.First(&Product{}, "code = ?", "after_delete_error").Error; err != nil {
		t.Errorf("Record shouldn't be deleted because of an error happened in after delete callback", err)
	}
}

func TestFillSmallerStructCorrectly(t *testing.T) {
	type SimpleUser struct {
		Name      string
		Id        int64
		UpdatedAt time.Time
		CreatedAt time.Time
	}

	var simple_user SimpleUser
	db.Table("users").Find(&simple_user)

	if simple_user.Id == 0 || simple_user.Name == "" {
		t.Errorf("Should fill data correctly even some column missing")
	}
}

func TestExceptionsWithInvalidSql(t *testing.T) {
	var columns []string
	if db.Where("sdsd.zaaa = ?", "sd;;;aa").Pluck("aaa", &columns).Error == nil {
		t.Errorf("Should got error with invalid SQL")
	}

	if db.Model(&User{}).Where("sdsd.zaaa = ?", "sd;;;aa").Pluck("aaa", &columns).Error == nil {
		t.Errorf("Should got error with invalid SQL")
	}

	type Article struct {
		Name string
	}
	db.Where("sdsd.zaaa = ?", "sd;;;aa").Find(&Article{})

	db.Where("name = ?", "3").Find(&[]User{})

	var count1, count2 int64
	db.Model(&User{}).Count(&count1)
	if count1 <= 0 {
		t.Errorf("Should find some users")
	}

	q := db.Where("name = ?", "jinzhu; delete * from users").First(&User{})
	if q.Error == nil {
		t.Errorf("Can't find user")
	}

	db.Model(&User{}).Count(&count2)
	if count1 != count2 {
		t.Errorf("Users should not be deleted by invalid SQL")
	}

	db.Where("non-existing = ?", "3").Find(&[]User{})
}

func TestSetTableDirectly(t *testing.T) {
	var ages []int64
	if db.Table("users").Pluck("age", &ages).Error != nil {
		t.Errorf("No errors should happen if set table for pluck")
	}

	if len(ages) == 0 {
		t.Errorf("Should get some records")
	}

	var users []User
	if db.Table("users").Find(&users).Error != nil {
		t.Errorf("No errors should happen if set table for find")
	}

	if db.Table("unexisting_users_table").Find(&users).Error == nil {
		t.Errorf("Should got error if set table to an invalid table")
	}

	db.Exec("drop table deleted_users;")
	if db.Table("deleted_users").CreateTable(&User{}).Error != nil {
		t.Errorf("Should create table with specified table")
	}

	db.Table("deleted_users").Save(&User{Name: "DeletedUser"})

	var deleted_users []User
	db.Table("deleted_users").Find(&deleted_users)
	if len(deleted_users) != 1 {
		t.Errorf("Query from specified table")
	}

	var deleted_user User
	db.Table("deleted_users").First(&deleted_user)
	if deleted_user.Name != "DeletedUser" {
		t.Errorf("Query from specified table")
	}

	var user1, user2, user3 User
	db.First(&user1).Table("deleted_users").First(&user2).Table("").First(&user3)
	if (user1.Name == user2.Name) || (user1.Name != user3.Name) {
		t.Errorf("unset specified table with blank string")
	}
}

func TestUpdate(t *testing.T) {
	product1 := Product{Code: "123"}
	product2 := Product{Code: "234"}
	animal1 := Animal{Name: "Ferdinand"}
	animal2 := Animal{Name: "nerdz"}

	db.Save(&product1).Save(&product2).Update("code", "456")

	if product2.Code != "456" {
		t.Errorf("Record should be updated with update attributes")
	}

	db.Save(&animal1).Save(&animal2).Update("name", "Francis")

	if animal2.Name != "Francis" {
		t.Errorf("Record should be updated with update attributes")
	}

	db.First(&product1, product1.Id)
	db.First(&product2, product2.Id)
	updated_at1 := product1.UpdatedAt
	updated_at2 := product2.UpdatedAt

	db.First(&animal1, animal1.Counter)
	db.First(&animal2, animal2.Counter)
	animalUpdated_at1 := animal1.UpdatedAt
	animalUpdated_at2 := animal2.UpdatedAt

	var product3 Product
	db.First(&product3, product2.Id).Update("code", "456")
	if updated_at2.Format(time.RFC3339Nano) != product3.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should not be updated if nothing changed")
	}

	if db.First(&Product{}, "code = '123'").Error != nil {
		t.Errorf("Product 123 should not be updated")
	}

	if db.First(&Product{}, "code = '234'").Error == nil {
		t.Errorf("Product 234 should be changed to 456")
	}

	if db.First(&Product{}, "code = '456'").Error != nil {
		t.Errorf("Product 234 should be changed to 456")
	}

	var animal3 Animal
	db.First(&animal3, animal2.Counter).Update("Name", "Robert")

	if animalUpdated_at2.Format(time.RFC3339Nano) != animal2.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should not be updated if nothing changed")
	}

	if db.First(&Animal{}, "name = 'Ferdinand'").Error != nil {
		t.Errorf("Animal 'Ferdinand' should not be updated")
	}

	if db.First(&Animal{}, "name = 'nerdz'").Error == nil {
		t.Errorf("Animal 'nerdz' should be changed to 'Francis'")
	}

	if db.First(&Animal{}, "name = 'Robert'").Error != nil {
		t.Errorf("Animal 'nerdz' should be changed to 'Robert'")
	}

	db.Table("products").Where("code in (?)", []string{"123"}).Update("code", "789")

	var product4 Product
	db.First(&product4, product1.Id)
	if updated_at1.Format(time.RFC3339Nano) != product4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should be updated if something changed")
	}

	if db.First(&Product{}, "code = '123'").Error == nil {
		t.Errorf("Product 123 should be changed to 789")
	}

	if db.First(&Product{}, "code = '456'").Error != nil {
		t.Errorf("Product 456 should not be changed to 789")
	}

	if db.First(&Product{}, "code = '789'").Error != nil {
		t.Errorf("Product 456 should be changed to 789")
	}

	if db.Model(product2).Update("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update with CamelCase")
	}

	if db.Model(&product2).UpdateColumn("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update_column with CamelCase")
	}

	db.Table("animals").Where("name in (?)", []string{"Ferdinand"}).Update("name", "Franz")

	var animal4 Animal
	db.First(&animal4, animal1.Counter)
	if animalUpdated_at1.Format(time.RFC3339Nano) != animal4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("animalUpdated_at should be updated if something changed")
	}

	if db.First(&Animal{}, "name = 'Ferdinand'").Error == nil {
		t.Errorf("Animal 'Fredinand' should be changed to 'Franz'")
	}

	if db.First(&Animal{}, "name = 'Robert'").Error != nil {
		t.Errorf("Animal 'Robert' should not be changed to 'Francis'")
	}

	if db.First(&Animal{}, "name = 'Franz'").Error != nil {
		t.Errorf("Product 'nerdz' should be changed to 'Franz'")
	}

	if db.Model(animal2).Update("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update with CamelCase")
	}

	if db.Model(&animal2).UpdateColumn("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update_column with CamelCase")
	}

	var animals []Animal
	db.Find(&animals)
	if count := db.Model(Animal{}).Update("CreatedAt", time.Now().Add(2*time.Hour)).RowsAffected; count != int64(len(animals)) {
		t.Error("RowsAffected should be correct when do batch update")
	}
}

func TestUpdates(t *testing.T) {
	product1 := Product{Code: "abc", Price: 10}
	product2 := Product{Code: "cde", Price: 20}
	db.Save(&product1).Save(&product2)
	db.Model(&product2).Updates(map[string]interface{}{"code": "edf", "price": 100})
	if product2.Code != "edf" || product2.Price != 100 {
		t.Errorf("Record should be updated also with update attributes")
	}

	db.First(&product1, product1.Id)
	db.First(&product2, product2.Id)
	updated_at1 := product1.UpdatedAt
	updated_at2 := product2.UpdatedAt

	var product3 Product
	db.First(&product3, product2.Id).Updates(Product{Code: "edf", Price: 100})
	if product3.Code != "edf" || product3.Price != 100 {
		t.Errorf("Record should be updated with update attributes")
	}

	if updated_at2.Format(time.RFC3339Nano) != product3.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should not be updated if nothing changed")
	}

	if db.First(&Product{}, "code = 'abc' and price = 10").Error != nil {
		t.Errorf("Product abc should not be updated")
	}

	if db.First(&Product{}, "code = 'cde'").Error == nil {
		t.Errorf("Product cde should be renamed to edf")
	}

	if db.First(&Product{}, "code = 'edf' and price = 100").Error != nil {
		t.Errorf("Product cde should be renamed to edf")
	}

	db.Table("products").Where("code in (?)", []string{"abc"}).Updates(map[string]string{"code": "fgh", "price": "200"})
	if db.First(&Product{}, "code = 'abc'").Error == nil {
		t.Errorf("Product abc's code should be changed to fgh")
	}

	var product4 Product
	db.First(&product4, product1.Id)
	if updated_at1.Format(time.RFC3339Nano) != product4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should be updated if something changed")
	}

	if db.First(&Product{}, "code = 'edf' and price = ?", 100).Error != nil {
		t.Errorf("Product cde's code should not be changed to fgh")
	}

	if db.First(&Product{}, "code = 'fgh' and price = 200").Error != nil {
		t.Errorf("We should have Product fgh")
	}
}

func TestUpdateColumn(t *testing.T) {
	product1 := Product{Code: "update_column 1", Price: 10}
	product2 := Product{Code: "update_column 2", Price: 20}
	db.Save(&product1).Save(&product2).UpdateColumn(map[string]interface{}{"code": "update_column 3", "price": 100})
	if product2.Code != "update_column 3" || product2.Price != 100 {
		t.Errorf("product 2 should be updated with update column")
	}

	var product3 Product
	db.First(&product3, product1.Id)
	if product3.Code != "update_column 1" || product3.Price != 10 {
		t.Errorf("product 1 should not be updated")
	}

	var product4, product5 Product
	db.First(&product4, product2.Id)
	updated_at1 := product4.UpdatedAt

	db.Model(Product{}).Where(product2.Id).UpdateColumn("code", "update_column_new")
	db.First(&product5, product2.Id)
	if updated_at1.Format(time.RFC3339Nano) != product5.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should not be updated with update column")
	}
}

func TestSoftDelete(t *testing.T) {
	type Order struct {
		Id        int64
		Amount    int64
		DeletedAt time.Time
	}
	db.Exec("drop table orders;")
	db.CreateTable(&Order{})

	order := Order{Amount: 1234}
	db.Save(&order)
	if db.First(&Order{}, "amount = ?", 1234).Error != nil {
		t.Errorf("No errors should happen when save an order")
	}

	db.Delete(&order)
	if db.First(&Order{}, "amount = 1234").Error == nil {
		t.Errorf("Can't find order because it is soft deleted")
	}

	if db.Unscoped().First(&Order{}, "amount = 1234").Error != nil {
		t.Errorf("Should be able to find out the soft deleted order with unscoped")
	}

	db.Unscoped().Delete(&order)
	if db.Unscoped().First(&Order{}, "amount = 1234").Error == nil {
		t.Errorf("Can't find permanently deleted order")
	}
}

func TestFindOrInitialize(t *testing.T) {
	var user1, user2, user3, user4, user5, user6 User
	db.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user1)
	if user1.Name != "find or init" || user1.Id != 0 || user1.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	db.Where(User{Name: "find or init", Age: 33}).FirstOrInit(&user2)
	if user2.Name != "find or init" || user2.Id != 0 || user2.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	db.FirstOrInit(&user3, map[string]interface{}{"name": "find or init 2"})
	if user3.Name != "find or init 2" || user3.Id != 0 {
		t.Errorf("user should be initialized with inline search value")
	}

	db.Where(&User{Name: "find or init"}).Attrs(User{Age: 44}).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.Id != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and attrs")
	}

	db.Where(&User{Name: "find or init"}).Assign("age", 44).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.Id != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and assign attrs")
	}

	db.Save(&User{Name: "find or init", Age: 33})
	db.Where(&User{Name: "find or init"}).Attrs("age", 44).FirstOrInit(&user5)
	if user5.Name != "find or init" || user5.Id == 0 || user5.Age != 33 {
		t.Errorf("user should be found and not initialized by Attrs")
	}

	db.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.Id == 0 || user6.Age != 33 {
		t.Errorf("user should be found with FirstOrInit")
	}

	db.Where(&User{Name: "find or init"}).Assign(User{Age: 44}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.Id == 0 || user6.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}
}

func TestFindOrCreate(t *testing.T) {
	var user1, user2, user3, user4, user5, user6, user7, user8 User
	db.Where(&User{Name: "find or create", Age: 33}).FirstOrCreate(&user1)
	if user1.Name != "find or create" || user1.Id == 0 || user1.Age != 33 {
		t.Errorf("user should be created with search value")
	}

	db.Where(&User{Name: "find or create", Age: 33}).FirstOrCreate(&user2)
	if user1.Id != user2.Id || user2.Name != "find or create" || user2.Id == 0 || user2.Age != 33 {
		t.Errorf("user should be created with search value")
	}

	db.FirstOrCreate(&user3, map[string]interface{}{"name": "find or create 2"})
	if user3.Name != "find or create 2" || user3.Id == 0 {
		t.Errorf("user should be created with inline search value")
	}

	db.Where(&User{Name: "find or create 3"}).Attrs("age", 44).FirstOrCreate(&user4)
	if user4.Name != "find or create 3" || user4.Id == 0 || user4.Age != 44 {
		t.Errorf("user should be created with search value and attrs")
	}

	updated_at1 := user4.UpdatedAt
	db.Where(&User{Name: "find or create 3"}).Assign("age", 55).FirstOrCreate(&user4)
	if updated_at1.Format(time.RFC3339Nano) == user4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("UpdateAt should be changed when update values with assign")
	}

	db.Where(&User{Name: "find or create 4"}).Assign(User{Age: 44}).FirstOrCreate(&user4)
	if user4.Name != "find or create 4" || user4.Id == 0 || user4.Age != 44 {
		t.Errorf("user should be created with search value and assigned attrs")
	}

	db.Where(&User{Name: "find or create"}).Attrs("age", 44).FirstOrInit(&user5)
	if user5.Name != "find or create" || user5.Id == 0 || user5.Age != 33 {
		t.Errorf("user should be found and not initialized by Attrs")
	}

	db.Where(&User{Name: "find or create"}).Assign(User{Age: 44}).FirstOrCreate(&user6)
	if user6.Name != "find or create" || user6.Id == 0 || user6.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}

	db.Where(&User{Name: "find or create"}).Find(&user7)
	if user7.Name != "find or create" || user7.Id == 0 || user7.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}

	db.Where(&User{Name: "find or create embedded struct"}).Assign(User{Age: 44, CreditCard: CreditCard{Number: "1231231231"}, Emails: []Email{{Email: "jinzhu@assign_embedded_struct.com"}, {Email: "jinzhu-2@assign_embedded_struct.com"}}}).FirstOrCreate(&user8)
	if db.Where("email = ?", "jinzhu-2@assign_embedded_struct.com").First(&Email{}).RecordNotFound() {
		t.Errorf("embedded struct email should be saved")
	}

	if db.Where("email = ?", "1231231231").First(&CreditCard{}).RecordNotFound() {
		t.Errorf("embedded struct credit card should be saved")
	}
}

func TestNot(t *testing.T) {
	var users1, users2, users3, users4, users5, users6, users7, users8 []User
	db.Find(&users1)

	db.Not(users1[0].Id).Find(&users2)

	if len(users1)-len(users2) != 1 {
		t.Errorf("Should ignore the first users with Not")
	}

	db.Not([]int{}).Find(&users3)
	if len(users1)-len(users3) != 0 {
		t.Errorf("Should find all users with a blank condition")
	}

	var name_3_count int64
	db.Table("users").Where("name = ?", "3").Count(&name_3_count)
	db.Not("name", "3").Find(&users4)
	if len(users1)-len(users4) != int(name_3_count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	users4 = []User{}
	db.Not("name = ?", "3").Find(&users4)
	if len(users1)-len(users4) != int(name_3_count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	users4 = []User{}
	db.Not("name <> ?", "3").Find(&users4)
	if len(users4) != int(name_3_count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	db.Not(User{Name: "3"}).Find(&users5)

	if len(users1)-len(users5) != int(name_3_count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	db.Not(map[string]interface{}{"name": "3"}).Find(&users6)
	if len(users1)-len(users6) != int(name_3_count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	db.Not("name", []string{"3"}).Find(&users7)
	if len(users1)-len(users7) != int(name_3_count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	var name_2_count int64
	db.Table("users").Where("name = ?", "2").Count(&name_2_count)
	db.Not("name", []string{"3", "2"}).Find(&users8)
	if len(users1)-len(users8) != (int(name_3_count) + int(name_2_count)) {
		t.Errorf("Should find all users's name not equal 3")
	}
}

type Category struct {
	Id   int64
	Name string
}

type Post struct {
	Id             int64
	CategoryId     sql.NullInt64
	MainCategoryId int64
	Title          string
	Body           string
	Comments       []Comment
	Category       Category
	MainCategory   Category
}

type Comment struct {
	Id      int64
	PostId  int64
	Content string
	Post    Post
}

func TestSubStruct(t *testing.T) {
	db.DropTable(Category{})
	db.DropTable(Post{})
	db.DropTable(Comment{})

	db.CreateTable(Category{})
	db.CreateTable(Post{})
	db.CreateTable(Comment{})

	post := Post{
		Title:        "post 1",
		Body:         "body 1",
		Comments:     []Comment{{Content: "Comment 1"}, {Content: "Comment 2"}},
		Category:     Category{Name: "Category 1"},
		MainCategory: Category{Name: "Main Category 1"},
	}

	if err := db.Save(&post).Error; err != nil {
		t.Errorf("Got errors when save post", err)
	}

	if db.First(&Category{}, "name = ?", "Category 1").Error != nil {
		t.Errorf("Category should be saved")
	}

	var p Post
	db.First(&p, post.Id)

	if post.CategoryId.Int64 == 0 || p.CategoryId.Int64 == 0 || post.MainCategoryId == 0 || p.MainCategoryId == 0 {
		t.Errorf("Category Id should exist")
	}

	if db.First(&Comment{}, "content = ?", "Comment 1").Error != nil {
		t.Errorf("Comment 1 should be saved")
	}
	if post.Comments[0].PostId == 0 {
		t.Errorf("Comment Should have post id")
	}

	var comment Comment
	if db.First(&comment, "content = ?", "Comment 2").Error != nil {
		t.Errorf("Comment 2 should be saved")
	}

	if comment.PostId == 0 {
		t.Errorf("Comment 2 Should have post id")
	}

	comment3 := Comment{Content: "Comment 3", Post: Post{Title: "Title 3", Body: "Body 3"}}
	db.Save(&comment3)
}

func TestIgnoreAssociation(t *testing.T) {
	user := User{Name: "ignore", IgnoredEmbedStruct: IgnoredEmbedStruct{Name: "IgnoreMe"}}
	if err := db.Save(&user).Error; err != nil {
		t.Errorf("Should have no error with ignored association, but got ", err)
	}
}

func TestRelated(t *testing.T) {
	user := User{
		Name:            "jinzhu",
		BillingAddress:  Address{Address1: "Billing Address - Address 1"},
		ShippingAddress: Address{Address1: "Shipping Address - Address 1"},
		Emails:          []Email{{Email: "jinzhu@example.com"}, {Email: "jinzhu-2@example@example.com"}},
		CreditCard:      CreditCard{Number: "1234567890"},
	}

	db.Save(&user)

	if user.CreditCard.Id == 0 {
		t.Errorf("After user save, credit card should have id")
	}

	if user.BillingAddress.Id == 0 {
		t.Errorf("After user save, billing address should have id")
	}

	if user.Emails[0].Id == 0 {
		t.Errorf("After user save, billing address should have id")
	}

	var emails []Email
	db.Model(&user).Related(&emails)
	if len(emails) != 2 {
		t.Errorf("Should have two emails")
	}

	var emails2 []Email
	db.Model(&user).Where("email = ?", "jinzhu@example.com").Related(&emails2)
	if len(emails2) != 1 {
		t.Errorf("Should have two emails")
	}

	var user1 User
	db.Model(&user).Related(&user1.Emails)
	if len(user1.Emails) != 2 {
		t.Errorf("Should have only one email match related condition")
	}

	var address1 Address
	db.Model(&user).Related(&address1, "BillingAddressId")
	if address1.Address1 != "Billing Address - Address 1" {
		t.Errorf("Should get billing address from user correctly")
	}

	user1 = User{}
	db.Model(&address1).Related(&user1, "BillingAddressId")
	if db.NewRecord(user1) {
		t.Errorf("Should get user from address correctly")
	}

	var user2 User
	db.Model(&emails[0]).Related(&user2)
	if user2.Id != user.Id || user2.Name != user.Name {
		t.Errorf("Should get user from email correctly")
	}

	var credit_card CreditCard
	var user3 User
	db.First(&credit_card, "number = ?", "1234567890")
	db.Model(&credit_card).Related(&user3)
	if user3.Id != user.Id || user3.Name != user.Name {
		t.Errorf("Should get user from credit card correctly")
	}

	if !db.Model(&CreditCard{}).Related(&User{}).RecordNotFound() {
		t.Errorf("RecordNotFound for Related")
	}
}

type Order struct {
}

type Cart struct {
}

func (c Cart) TableName() string {
	return "shopping_cart"
}

func TestTableName(t *testing.T) {
	if db.NewScope(Order{}).TableName() != "orders" {
		t.Errorf("Order's table name should be orders")
	}

	if db.NewScope(&Order{}).TableName() != "orders" {
		t.Errorf("&Order's table name should be orders")
	}

	if db.NewScope([]Order{}).TableName() != "orders" {
		t.Errorf("[]Order's table name should be orders")
	}

	if db.NewScope(&[]Order{}).TableName() != "orders" {
		t.Errorf("&[]Order's table name should be orders")
	}

	db.SingularTable(true)
	if db.NewScope(Order{}).TableName() != "order" {
		t.Errorf("Order's singular table name should be order")
	}

	if db.NewScope(&Order{}).TableName() != "order" {
		t.Errorf("&Order's singular table name should be order")
	}

	if db.NewScope([]Order{}).TableName() != "order" {
		t.Errorf("[]Order's singular table name should be order")
	}

	if db.NewScope(&[]Order{}).TableName() != "order" {
		t.Errorf("&[]Order's singular table name should be order")
	}

	if db.NewScope(&Cart{}).TableName() != "shopping_cart" {
		t.Errorf("&Cart's singular table name should be shopping_cart")
	}

	if db.NewScope(Cart{}).TableName() != "shopping_cart" {
		t.Errorf("Cart's singular table name should be shopping_cart")
	}

	if db.NewScope(&[]Cart{}).TableName() != "shopping_cart" {
		t.Errorf("&[]Cart's singular table name should be shopping_cart")
	}

	if db.NewScope([]Cart{}).TableName() != "shopping_cart" {
		t.Errorf("[]Cart's singular table name should be shopping_cart")
	}
	db.SingularTable(false)
}

type BigEmail struct {
	Id           int64
	UserId       int64
	Email        string
	UserAgent    string
	RegisteredAt time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (b BigEmail) TableName() string {
	return "emails"
}

func TestAutoMigration(t *testing.T) {
	db.AutoMigrate(Address{})
	if err := db.Table("emails").AutoMigrate(BigEmail{}).Error; err != nil {
		t.Errorf("Auto Migrate should not raise any error", err)
	}

	db.Save(&BigEmail{Email: "jinzhu@example.org", UserAgent: "pc", RegisteredAt: time.Now()})

	var big_email BigEmail
	db.First(&big_email, "user_agent = ?", "pc")
	if big_email.Email != "jinzhu@example.org" || big_email.UserAgent != "pc" || big_email.RegisteredAt.IsZero() {
		t.Error("Big Emails should be saved and fetched correctly")
	}

	if err := db.Save(&BigEmail{Email: "jinzhu@example.org", UserAgent: "pc", RegisteredAt: time.Now()}).Error; err == nil {
		t.Error("Should not be able to save because of unique tag")
	}
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

type NullValue struct {
	Id      int64
	Name    sql.NullString `sql:"not null"`
	Age     sql.NullInt64
	Male    sql.NullBool
	Height  sql.NullFloat64
	AddedAt NullTime
}

func TestSqlNullValue(t *testing.T) {
	db.DropTable(&NullValue{})
	db.AutoMigrate(&NullValue{})

	if err := db.Save(&NullValue{Name: sql.NullString{"hello", true}, Age: sql.NullInt64{18, true}, Male: sql.NullBool{true, true}, Height: sql.NullFloat64{100.11, true}, AddedAt: NullTime{time.Now(), true}}).Error; err != nil {
		t.Errorf("Not error should raise when test null value", err)
	}

	var nv NullValue
	db.First(&nv, "name = ?", "hello")

	if nv.Name.String != "hello" || nv.Age.Int64 != 18 || nv.Male.Bool != true || nv.Height.Float64 != 100.11 || nv.AddedAt.Valid != true {
		t.Errorf("Should be able to fetch null value")
	}

	if err := db.Save(&NullValue{Name: sql.NullString{"hello-2", true}, Age: sql.NullInt64{18, false}, Male: sql.NullBool{true, true}, Height: sql.NullFloat64{100.11, true}, AddedAt: NullTime{time.Now(), false}}).Error; err != nil {
		t.Errorf("Not error should raise when test null value", err)
	}

	var nv2 NullValue
	db.First(&nv2, "name = ?", "hello-2")
	if nv2.Name.String != "hello-2" || nv2.Age.Int64 != 0 || nv2.Male.Bool != true || nv2.Height.Float64 != 100.11 || nv2.AddedAt.Valid != false {
		t.Errorf("Should be able to fetch null value")
	}

	if err := db.Save(&NullValue{Name: sql.NullString{"hello-3", false}, Age: sql.NullInt64{18, false}, Male: sql.NullBool{true, true}, Height: sql.NullFloat64{100.11, true}, AddedAt: NullTime{time.Now(), false}}).Error; err == nil {
		t.Errorf("Can't save because of name can't be null", err)
	}
}

func TestTransaction(t *testing.T) {
	tx := db.Begin()
	u := User{Name: "transcation"}
	if err := tx.Save(&u).Error; err != nil {
		t.Errorf("No error should raise, but got", err)
	}

	if err := tx.First(&User{}, "name = ?", "transcation").Error; err != nil {
		t.Errorf("Should find saved record, but got", err)
	}

	if sql_tx, ok := tx.CommonDB().(*sql.Tx); !ok || sql_tx == nil {
		t.Errorf("Should return the underlying sql.Tx")
	}

	tx.Rollback()

	if err := tx.First(&User{}, "name = ?", "transcation").Error; err == nil {
		t.Errorf("Should not find record after rollback")
	}

	tx2 := db.Begin()
	u2 := User{Name: "transcation-2"}
	if err := tx2.Save(&u2).Error; err != nil {
		t.Errorf("No error should raise, but got", err)
	}

	if err := tx2.First(&User{}, "name = ?", "transcation-2").Error; err != nil {
		t.Errorf("Should find saved record, but got", err)
	}

	tx2.Commit()

	if err := db.First(&User{}, "name = ?", "transcation-2").Error; err != nil {
		t.Errorf("Should be able to find committed record")
	}
}

func TestQueryChain(t *testing.T) {
	var user_count1, user_count2 int64
	d := db.Model(User{}).Where("age > ?", 20)
	d.Where("name = ?", "3").Count(&user_count1)
	d.Count(&user_count2)
	if user_count2 == user_count1 {
		t.Error("DB object should be cloned when search")
	}
}

func TestRow(t *testing.T) {
	row := db.Table("users").Where("name = ?", "2").Select("age").Row()
	var age int64
	row.Scan(&age)
	if age != 20 {
		t.Errorf("Scan with Row")
	}
}

func TestRows(t *testing.T) {
	rows, err := db.Table("users").Where("name = ?", "3").Select("name, age").Rows()
	if err != nil {
		t.Errorf("Not error should happen, but got", err)
	}

	count := 0
	for rows.Next() {
		var name string
		var age int64
		rows.Scan(&name, &age)
		count++
	}
	if count != 2 {
		t.Errorf("Should found two records with name 3")
	}
}

type result struct {
	Name string
	Age  int
}

func TestScan(t *testing.T) {
	var res result
	db.Table("users").Select("name, age").Where("name = ?", 3).Scan(&res)
	if res.Name != "3" {
		t.Errorf("Scan into struct should work")
	}

	var doubleAgeRes result
	db.Table("users").Select("age + age as age").Where("name = ?", 3).Scan(&doubleAgeRes)
	if doubleAgeRes.Age != res.Age*2 {
		t.Errorf("Scan double age as age")
	}

	var ress []result
	db.Table("users").Select("name, age").Where("name = ?", 3).Scan(&ress)
	if len(ress) != 2 || ress[0].Name != "3" || ress[1].Name != "3" {
		t.Errorf("Scan into struct map")
	}
}

func TestRaw(t *testing.T) {
	var ress []result
	db.Raw("SELECT name, age FROM users WHERE name = ?", 3).Scan(&ress)
	if len(ress) != 2 || ress[0].Name != "3" || ress[1].Name != "3" {
		t.Errorf("Raw with scan")
	}

	rows, _ := db.Raw("select name, age from users where name = ?", 3).Rows()
	count := 0
	for rows.Next() {
		count++
	}
	if count != 2 {
		t.Errorf("Raw with Rows should find two records with name 3")
	}
}

func TestGroup(t *testing.T) {
	rows, err := db.Select("name").Table("users").Group("name").Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			rows.Scan(&name)
		}
	} else {
		t.Errorf("Should not raise any error")
	}
}

func TestJoins(t *testing.T) {
	type result struct {
		Name  string
		Email string
	}

	user := User{
		Name:   "joins",
		Emails: []Email{{Email: "join1@example.com"}, {Email: "join2@example.com"}},
	}
	db.Save(&user)

	var results []result
	db.Table("users").Select("name, email").Joins("left join emails on emails.user_id = users.id").Where("name = ?", "joins").Scan(&results)
	if len(results) != 2 || results[0].Email != "join1@example.com" || results[1].Email != "join2@example.com" {
		t.Errorf("Should find all two emails with Join")
	}
}

func NameIn1And2(d *gorm.DB) *gorm.DB {
	return d.Where("name in (?)", []string{"1", "2"})
}

func NameIn2And3(d *gorm.DB) *gorm.DB {
	return d.Where("name in (?)", []string{"2", "3"})
}

func NameIn(names []string) func(d *gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		return d.Where("name in (?)", names)
	}
}

func TestScopes(t *testing.T) {
	var users1, users2, users3 []User
	db.Scopes(NameIn1And2).Find(&users1)
	if len(users1) != 2 {
		t.Errorf("Should only have two users's name in 1, 2")
	}

	db.Scopes(NameIn1And2, NameIn2And3).Find(&users2)
	if len(users2) != 1 {
		t.Errorf("Should only have two users's name is 2")
	}

	db.Scopes(NameIn([]string{"1", "2"})).Find(&users3)
	if len(users3) != 2 {
		t.Errorf("Should only have two users's name is 2")
	}
}

func TestHaving(t *testing.T) {
	rows, err := db.Select("name, count(*) as total").Table("users").Group("name").Having("name IN (?)", []string{"2", "3"}).Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			var total int64
			rows.Scan(&name, &total)

			if name == "2" && total != 1 {
				t.Errorf("Should have one user having name 2", total)
			}
			if name == "3" && total != 2 {
				t.Errorf("Should have two users having name 3", total)
			}
		}
	} else {
		t.Errorf("Should not raise any error", err)
	}
}

func TestAnonymousField(t *testing.T) {
	user := User{Name: "anonymous_field", Company: Company{Name: "company"}}
	db.Save(&user)

	var user2 User
	db.First(&user2, "name = ?", "anonymous_field")
	db.Model(&user2).Related(&user2.Company)
	if user2.Company.Name != "company" {
		t.Errorf("Should be able to get anonymous field")
	}
}

func TestAnonymousScanner(t *testing.T) {
	user := User{Name: "anonymous_scanner", Role: Role{Name: "admin"}}
	db.Save(&user)

	var user2 User
	db.First(&user2, "name = ?", "anonymous_scanner")
	if user2.Role.Name != "admin" {
		t.Errorf("Should be able to get anonymous scanner")
	}

	if !user2.IsAdmin() {
		t.Errorf("Should be able to get anonymous scanner")
	}
}

func TestExecRawSql(t *testing.T) {
	db.Exec("update users set name=? where name in (?)", "jinzhu", []string{"1", "2", "3"})
	if db.Where("name in (?)", []string{"1", "2", "3"}).First(&User{}).Error != gorm.RecordNotFound {
		t.Error("Raw sql should be able to parse argument")
	}
}

func TestTimeWithZone(t *testing.T) {
	var format = "2006-01-02 15:04:05 -0700"
	var times []time.Time
	GMT8, _ := time.LoadLocation("Asia/Shanghai")
	times = append(times, time.Date(2013, 02, 19, 1, 51, 49, 123456789, GMT8))
	times = append(times, time.Date(2013, 02, 18, 17, 51, 49, 123456789, time.UTC))

	for index, vtime := range times {
		name := "time_with_zone_" + strconv.Itoa(index)
		user := User{Name: name, Birthday: vtime}
		db.Save(&user)
		if user.Birthday.UTC().Format(format) != "2013-02-18 17:51:49 +0000" {
			t.Errorf("User's birthday should not be changed after save")
		}

		if user.DeletedAt.UTC().Format(format) != "0001-01-01 00:00:00 +0000" {
			t.Errorf("User's deleted at should be zero")
		}

		var findUser, findUser2, findUser3 User
		db.First(&findUser, "name = ?", name)
		if findUser.Birthday.UTC().Format(format) != "2013-02-18 17:51:49 +0000" {
			t.Errorf("User's birthday should not be changed after find")
		}

		if findUser.DeletedAt.UTC().Format(format) != "0001-01-01 00:00:00 +0000" {
			t.Errorf("User's deleted at should be zero")
		}

		if db.Where("birthday >= ?", vtime.Add(-time.Minute)).First(&findUser2).RecordNotFound() {
			t.Errorf("User should be found")
		}

		if !db.Where("birthday >= ?", vtime.Add(time.Minute)).First(&findUser3).RecordNotFound() {
			t.Errorf("User should not be found")
		}
	}
}

func BenchmarkGorm(b *testing.B) {
	b.N = 2000
	for x := 0; x < b.N; x++ {
		e := strconv.Itoa(x) + "benchmark@example.org"
		email := BigEmail{Email: e, UserAgent: "pc", RegisteredAt: time.Now()}
		// Insert
		db.Save(&email)
		// Query
		db.First(&BigEmail{}, "email = ?", e)
		// Update
		db.Model(&email).UpdateColumn("email", "new-"+e)
		// Delete
		db.Delete(&email)
	}
}

func BenchmarkRawSql(b *testing.B) {
	db, _ := sql.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
	db.SetMaxIdleConns(10)
	insert_sql := "INSERT INTO emails (user_id,email,user_agent,registered_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id"
	query_sql := "SELECT * FROM emails WHERE email = $1 ORDER BY id LIMIT 1"
	update_sql := "UPDATE emails SET email = $1, updated_at = $2 WHERE id = $3"
	delete_sql := "DELETE FROM orders WHERE id = $1"

	b.N = 2000
	for x := 0; x < b.N; x++ {
		var id int64
		e := strconv.Itoa(x) + "benchmark@example.org"
		email := BigEmail{Email: e, UserAgent: "pc", RegisteredAt: time.Now()}
		// Insert
		db.QueryRow(insert_sql, email.UserId, email.Email, email.UserAgent, email.RegisteredAt, time.Now(), time.Now()).Scan(&id)
		// Query
		rows, _ := db.Query(query_sql, email.Email)
		rows.Close()
		// Update
		db.Exec(update_sql, "new-"+e, time.Now(), id)
		// Delete
		db.Exec(delete_sql, id)
	}
}

func TestSelectWithEscapedFieldName(t *testing.T) {
	var names []string
	db.Model(Animal{}).Where(&Animal{From: "nerdz"}).Pluck("\"name\"", &names)

	if len(names) != 1 {
		t.Errorf("Expected one name, but got: %d", len(names))
	}
}

func TestIndices(t *testing.T) {
	if err := db.Model(&UserCompany{}).AddIndex("idx_user_company_user", "user_id").Error; err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}
	if err := db.Model(&UserCompany{}).RemoveIndex("idx_user_company_user").Error; err != nil {
		t.Errorf("Got error when tried to remove index: %+v", err)
	}

	if err := db.Model(&UserCompany{}).AddIndex("idx_user_company_user_company", "user_id", "company_id").Error; err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}
	if err := db.Model(&UserCompany{}).RemoveIndex("idx_user_company_user_company").Error; err != nil {
		t.Errorf("Got error when tried to remove index: %+v", err)
	}

	if err := db.Model(&UserCompany{}).AddUniqueIndex("idx_user_company_user_company", "user_id", "company_id").Error; err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}
}

func TestHstore(t *testing.T) {
	if dialect := os.Getenv("GORM_DIALECT"); dialect != "postgres" {
		t.Skip()
	}

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS hstore").Error; err != nil {
		fmt.Println("\033[31mHINT: Must be superuser to create hstore extension (ALTER USER gorm WITH SUPERUSER;)\033[0m")
		panic(fmt.Sprintf("No error should happen when create hstore extension, but got %+v", err))
	}

	db.Exec("drop table details")

	if err := db.CreateTable(&Details{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	bankAccountId, phoneNumber, opinion := "123456", "14151321232", "sharkbait"
	bulk := map[string]*string{
		"bankAccountId": &bankAccountId,
		"phoneNumber":   &phoneNumber,
		"opinion":       &opinion,
	}
	d := Details{Bulk: bulk}
	db.Save(&d)

	var d2 Details
	if err := db.First(&d2).Error; err != nil {
		t.Errorf("Got error when tried to fetch details: %+v", err)
	}

	for k := range bulk {
		r, ok := d2.Bulk[k]
		if !ok {
			t.Errorf("Details should be existed")
		}
		if res, _ := bulk[k]; *res != *r {
			t.Errorf("Details should be equal")
		}
	}

}

func TestCreate(t *testing.T) {
	if err := db.Create(&UserCompany{Id: 10, UserId: 1, CompanyId: 1}).Error; err != nil {
		t.Error("Should be able to create record with predefined Id")
	}

	if db.First(&UserCompany{}, 10).RecordNotFound() {
		t.Error("Record created with predefined primary key not found")
	}

	if err := db.Create(&UserCompany{Id: 10, UserId: 10, CompanyId: 10}).Error; err == nil {
		t.Error("Should not be able to create record with predefined duplicate Id")
	}
}

func TestCompatibilityMode(t *testing.T) {
	db, _ := gorm.Open("testdb", "")
	testdb.SetQueryFunc(func(query string) (driver.Rows, error) {
		columns := []string{"id", "name", "age"}
		result := `
		1,Tim,20
		2,Joe,25
		3,Bob,30
		`
		return testdb.RowsFromCSVString(columns, result), nil
	})

	var users []User
	db.Find(&users)
	if (users[0].Name != "Tim") || len(users) != 3 {
		t.Errorf("Unexcepted result returned")
	}
}
