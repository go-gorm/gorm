package gorm

import (
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"reflect"
	"strconv"

	"testing"
	"time"
)

type User struct {
	Id        int64
	Age       int64
	Birthday  time.Time
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Product struct {
	Id                    int64
	Code                  string
	Price                 int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
	BeforeCreateCallTimes int64
	AfterCreateCallTimes  int64
	BeforeUpdateCallTimes int64
	AfterUpdateCallTimes  int64
	BeforeSaveCallTimes   int64
	AfterSaveCallTimes    int64
	BeforeDeleteCallTimes int64
	AfterDeleteCallTimes  int64
}

var (
	db                 DB
	t1, t2, t3, t4, t5 time.Time
)

func init() {
	var err error
	db, err = Open("postgres", "user=gorm dbname=gorm sslmode=disable")
	if err != nil {
		panic(fmt.Sprintf("No error should happen when connect database, but got %+v", err))
	}
	db.SetPool(10)

	err = db.Exec("drop table users;").Error
	if err != nil {
		fmt.Printf("Got error when try to delete table users, %+v\n", err)
	}
	db.Exec("drop table products;")

	orm := db.CreateTable(&User{})
	if orm.Error != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", orm.Error))
	}

	db.CreateTable(&Product{})

	var shortForm = "2006-01-02 15:04:05"
	t1, _ = time.Parse(shortForm, "2000-10-27 12:02:40")
	t2, _ = time.Parse(shortForm, "2002-01-01 00:00:00")
	t3, _ = time.Parse(shortForm, "2005-01-01 00:00:00")
	t4, _ = time.Parse(shortForm, "2010-01-01 00:00:00")
	t5, _ = time.Parse(shortForm, "2020-01-01 00:00:00")
	db.Save(&User{Name: "1", Age: 18, Birthday: t1})
	db.Save(&User{Name: "2", Age: 20, Birthday: t2})
	db.Save(&User{Name: "3", Age: 22, Birthday: t3})
	db.Save(&User{Name: "3", Age: 24, Birthday: t4})
	db.Save(&User{Name: "5", Age: 26, Birthday: t4})
}

func TestSaveAndFind(t *testing.T) {
	name := "save_and_find"
	u := &User{Name: name, Age: 1}
	db.Save(u)
	if u.Id == 0 {
		t.Errorf("Should have ID after create record")
	}

	user := &User{}
	db.First(user, "name = ?", name)
	if user.Name != name {
		t.Errorf("User should be saved and fetched correctly")
	}

	users := []User{}
	db.Find(&users)
}

func TestSaveAndUpdate(t *testing.T) {
	name, name2, new_name := "update", "update2", "new_update"
	user := User{Name: name, Age: 1}
	db.Save(&user)
	db.Save(&User{Name: name2, Age: 1})

	if user.Id == 0 {
		t.Errorf("User Id should exist after create")
	}

	user.Name = new_name
	db.Save(&user)
	orm := db.Where("name = ?", name).First(&User{})
	if orm.Error == nil {
		t.Errorf("Should raise error when looking for a existing user with an outdated name")
	}

	orm = db.Where("name = ?", new_name).First(&User{})
	if orm.Error != nil {
		t.Errorf("Shouldn't raise error when looking for a existing user with the new name")
	}

	orm = db.Where("name = ?", name2).First(&User{})
	if orm.Error != nil {
		t.Errorf("Shouldn't update other users")
	}
}

func TestDelete(t *testing.T) {
	name, name2 := "delete", "delete2"
	user := User{Name: name, Age: 1}
	db.Save(&user)
	db.Save(&User{Name: name2, Age: 1})
	orm := db.Delete(&user)

	orm = db.Where("name = ?", name).First(&User{})
	if orm.Error == nil {
		t.Errorf("User should be deleted successfully")
	}

	orm = db.Where("name = ?", name2).First(&User{})
	if orm.Error != nil {
		t.Errorf("User2 should not be deleted")
	}
}

func TestWhere(t *testing.T) {
	name := "where"
	db.Save(&User{Name: name, Age: 1})

	user := &User{}
	db.Where("name = ?", name).First(user)
	if user.Name != name {
		t.Errorf("Should found out user with name '%v'", name)
	}

	if db.Where(user.Id).First(&User{}).Error != nil {
		t.Errorf("Should found out users only with id")
	}

	user = &User{}
	orm := db.Where("name LIKE ?", "%nonono%").First(user)
	if orm.Error == nil {
		t.Errorf("Should return error when searching for none existing record, %+v", user)
	}

	user = &User{}
	orm = db.Where("name LIKE ?", "%whe%").First(user)
	if orm.Error != nil {
		t.Errorf("Should not return error when searching for existing record, %+v", user)
	}

	user = &User{}
	orm = db.Where("name = ?", "noexisting-user").First(user)
	if orm.Error == nil {
		t.Errorf("Should return error when looking for none existing record, %+v", user)
	}

	users := []User{}
	orm = db.Where("name = ?", "none-noexisting").Find(&users)
	if orm.Error != nil {
		t.Errorf("Shouldn't return error when looking for none existing records, %+v", users)
	}
	if len(users) != 0 {
		t.Errorf("Shouldn't find anything when looking for none existing records, %+v", users)
	}
}

func TestComplexWhere(t *testing.T) {
	var users []User
	db.Where("age > ?", 20).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should only found 3 users that age > 20, but have %v", len(users))
	}

	var user_ids []int64
	db.Table("users").Where("age > ?", 20).Pluck("id", &user_ids)
	if len(user_ids) != 3 {
		t.Errorf("Should only found 3 users that age > 20, but have %v", len(users))
	}

	users = []User{}
	db.Where(user_ids).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should only found 3 users that age > 20 when search with id map, but have %v", len(users))
	}

	users = []User{}
	db.Where("age >= ?", 20).Find(&users)
	if len(users) != 4 {
		t.Errorf("Should only found 4 users that age >= 20, but have %v", len(users))
	}

	users = []User{}
	db.Where("age = ?", 20).Find(&users)
	if len(users) != 1 {
		t.Errorf("Should only found 1 users age == 20, but have %v", len(users))
	}

	users = []User{}
	db.Where("age <> ?", 20).Find(&users)
	if len(users) < 3 {
		t.Errorf("Should have more than 3 users age != 20, but have %v", len(users))
	}

	users = []User{}
	db.Where("name = ? and age >= ?", "3", 20).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should only found 2 users that age >= 20 with name 3, but have %v", len(users))
	}

	users = []User{}
	db.Where("name = ?", "3").Where("age >= ?", 20).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should only found 2 users that age >= 20 with name 3, but have %v", len(users))
	}

	users = []User{}
	db.Where("birthday > ?", t2).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should only found 3 users's birthday >= t2", len(users))
	}

	users = []User{}
	db.Where("birthday >= ?", t1).Where("birthday < ?", t2).Find(&users)
	if len(users) != 1 {
		t.Errorf("Should only found 1 users's birthday <= t2, but have %v", len(users))
	}

	users = []User{}
	db.Where("birthday >= ? and birthday <= ?", t1, t2).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should only found 2 users's birthday <= t2, but have %v", len(users))
	}

	users = []User{}
	db.Where("name in (?)", []string{"1", "3"}).Find(&users)

	if len(users) != 3 {
		t.Errorf("Should only found 3 users's name in (1, 3), but have %v", len(users))
	}

	user_ids = []int64{}
	for _, user := range users {
		user_ids = append(user_ids, user.Id)
	}
	users = []User{}
	db.Where("id in (?)", user_ids).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should only found 3 users's name in (1, 3) - search by id, but have %v", len(users))
	}

	users = []User{}
	db.Where("name in (?)", []string{"1", "2"}).Find(&users)

	if len(users) != 2 {
		t.Errorf("Should only found 2 users's name in (1, 2), but have %v", len(users))
	}

	user_ids = []int64{}
	for _, user := range users {
		user_ids = append(user_ids, user.Id)
	}
	users = []User{}
	db.Where("id in (?)", user_ids).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should only found 2 users's name in (1, 2) - search by id, but have %v", len(users))
	}

	users = []User{}
	db.Where("id in (?)", user_ids[0]).Find(&users)
	if len(users) != 1 {
		t.Errorf("Should only found 1 users's name in (1, 2) - search by the first id, but have %v", len(users))
	}
}

func TestWhereWithStruct(t *testing.T) {
	var user User
	db.First(&user, &User{Name: "2"})
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Should be able to search first record with inline struct")
	}

	db.First(&user, User{Name: "2"})
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Should be able to search first record with inline struct")
	}

	db.Where(&User{Name: "2"}).First(&user)
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Should be able to search first record with where struct")
	}

	var users []User
	db.Find(&users, &User{Name: "3"})
	if len(users) != 2 {
		t.Errorf("Should be able to search all record with inline struct")
	}

	db.Where(User{Name: "3"}).Find(&users)
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Should be able to search first record with where struct")
	}
}

func TestWhereWithInterfaceMap(t *testing.T) {
	var user User
	db.First(&user, map[string]interface{}{"name": "2"})
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Should be able to search first record with inline interface map")
	}

	db.Where(map[string]interface{}{"name": "2"}).First(&user)
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Should be able to search first record with where interface map")
	}

	var users []User
	db.Find(&users, map[string]interface{}{"name": "3"})
	if len(users) != 2 {
		t.Errorf("Should be able to search all record with inline interface map")
	}

	db.Where(map[string]interface{}{"name": "3"}).Find(&users)
	if user.Id == 0 || user.Name != "2" {
		t.Errorf("Should be able to search first record with where interface map")
	}
}

func TestInitlineCondition(t *testing.T) {
	var u1, u2, u3, u4, u5, u6, u7 User
	db.Where("name = ?", "3").Order("age desc").First(&u1).First(&u2)

	db.Where("name = ?", "3").First(&u3, "age = 22").First(&u4, "age = ?", 24).First(&u5, "age = ?", 26)
	if !((u5.Id == 0) && (u3.Age == 22 && u3.Name == "3") && (u4.Age == 24 && u4.Name == "3")) {
		t.Errorf("Inline where condition for first when search")
	}

	var us1, us2, us3, us4 []User
	db.Find(&us1, "age = 22").Find(&us2, "name = ?", "3").Find(&us3, "age > ?", 20)
	if !(len(us1) == 1 && len(us2) == 2 && len(us3) == 3) {
		t.Errorf("Inline where condition for find when search")
	}

	db.Find(&us4, "name = ? and age > ?", "3", "22")
	if len(us4) != 1 {
		t.Errorf("More complex inline where condition for find, %v", us4)
	}

	db.First(&u6, u1.Id)
	if !(u6.Id == u1.Id && u6.Id != 0) {
		t.Errorf("Should find out user with int id")
	}

	db.First(&u7, strconv.Itoa(int(u1.Id)))
	if !(u6.Id == u1.Id && u6.Id != 0) {
		t.Errorf("Should find out user with string id")
	}
}

func TestSelect(t *testing.T) {
	var user User
	db.Where("name = ?", "3").Select("name").Find(&user)
	if user.Id != 0 {
		t.Errorf("Should not got ID because I am only looking for age, %+v", user.Id)
	}
	if user.Name != "3" {
		t.Errorf("Should got Name = 3 when searching it, %+v", user.Id)
	}

	query := db.Where("name = ?", "3").Select("nam;e")
	if query.Error == nil {
		t.Errorf("Should got error with invalid select string")
	}
}

func TestOrderAndPluck(t *testing.T) {
	var ages []int64
	db.Model(&[]User{}).Order("age desc").Pluck("age", &ages)
	if ages[0] != 26 {
		t.Errorf("The first age should be 26 because of ordered by")
	}

	var ages1, ages2 []int64
	db.Model(&[]User{}).Order("age desc").Pluck("age", &ages1).Order("age").Pluck("age", &ages2)
	if !reflect.DeepEqual(ages1, ages2) {
		t.Errorf("The first order is the primary order")
	}

	var ages3, ages4 []int64
	db.Model(&[]User{}).Order("age desc").Pluck("age", &ages3).Order("age", true).Pluck("age", &ages4)
	if reflect.DeepEqual(ages3, ages4) {
		t.Errorf("Reorder should works well")
	}

	ages = []int64{}
	var names []string
	db.Model(&User{}).Order("name").Order("age desc").Pluck("age", &ages).Pluck("name", &names)
	if !(names[0] == "1" && names[2] == "3" && names[3] == "3" && ages[2] == 24 && ages[3] == 22) {
		t.Errorf("Should be ordered correctly with multiple orders")
	}
}

func TestLimit(t *testing.T) {
	var users1, users2, users3 []User
	db.Order("age desc").Limit(3).Find(&users1).Limit(5).Find(&users2).Limit(-1).Find(&users3)

	if !(len(users1) == 3 && len(users2) == 5 && len(users3) > 5) {
		t.Errorf("Limit should works perfectly")
	}
}

func TestOffset(t *testing.T) {
	var users1, users2, users3, users4 []User
	db.Order("age desc").Find(&users1).Offset(3).Find(&users2).Offset(5).Find(&users3).Offset(-1).Find(&users4)

	if !((len(users1) == len(users4)) && (len(users1)-len(users2) == 3) && (len(users1)-len(users3) == 5)) {
		t.Errorf("Offset should works perfectly")
	}
}

func TestOr(t *testing.T) {
	var users []User
	db.Where("name = ?", "1").Or("name = ?", "3").Find(&users)
	if len(users) != 3 {
		t.Errorf("Should find three users with name 1 and 3")
	}
}

func TestCount(t *testing.T) {
	var count, count1, count2 int64
	var users []User
	db.Where("name = ?", "1").Or("name = ?", "3").Find(&users).Count(&count)
	if count != int64(len(users)) {
		t.Errorf("Count() method should get same value of users count")
	}

	db.Model(&User{}).Where("name = ?", "1").Count(&count1).Or("name = ?", "3").Count(&count2)
	if !(count1 == 1 && count2 == 3) {
		t.Errorf("Multiple count should works well also")
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

func (s *Product) AfterCreate() {
	s.AfterCreateCallTimes = s.AfterCreateCallTimes + 1
}

func (s *Product) AfterUpdate() {
	s.AfterUpdateCallTimes = s.AfterUpdateCallTimes + 1
}

func (s *Product) AfterSave() {
	s.AfterSaveCallTimes = s.AfterSaveCallTimes + 1
}

func (s *Product) BeforeDelete() (err error) {
	if s.Code == "dont_delete" {
		err = errors.New("Can't delete")
	}
	s.BeforeDeleteCallTimes = s.BeforeDeleteCallTimes + 1
	return
}

func (s *Product) AfterDelete() {
	s.AfterDeleteCallTimes = s.AfterDeleteCallTimes + 1
}
func (p *Product) GetCallTimes() []int64 {
	return []int64{p.BeforeCreateCallTimes, p.BeforeSaveCallTimes, p.BeforeUpdateCallTimes, p.AfterCreateCallTimes, p.AfterSaveCallTimes, p.AfterUpdateCallTimes, p.BeforeDeleteCallTimes, p.AfterDeleteCallTimes}
}

func TestRunCallbacks(t *testing.T) {
	p := Product{Code: "unique_code", Price: 100}
	db.Save(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 1, 0, 1, 1, 0, 0, 0}) {
		t.Errorf("Some errors happened when run create callbacks, %v", p.GetCallTimes())
	}

	db.Where("Code = ?", "unique_code").First(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 1, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("Should be able to query about saved values in before filters, %v", p.GetCallTimes())
	}

	p.Price = 200
	db.Save(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 2, 1, 0, 1, 1, 0, 0}) {
		t.Errorf("Some errors happened when run update callbacks, %v", p.GetCallTimes())
	}

	db.Where("Code = ?", "unique_code").First(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 2, 1, 0, 0, 0, 0, 0}) {
		t.Errorf("Some errors happened when run update callbacks, %v", p.GetCallTimes())
	}

	db.Delete(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 2, 1, 0, 0, 0, 1, 1}) {
		t.Errorf("Some errors happened when run update callbacks, %v", p.GetCallTimes())
	}

	if db.Where("Code = ?", "unique_code").First(&p).Error == nil {
		t.Errorf("Should get error when find an deleted record")
	}
}

func TestRunCallbacksAndGetErrors(t *testing.T) {
	p := Product{Code: "Invalid", Price: 100}
	if db.Save(&p).Error == nil {
		t.Errorf("An error from create callbacks expected when create")
	}

	if db.Where("code = ?", "Invalid").First(&Product{}).Error == nil {
		t.Errorf("Should not save records that have errors")
	}

	if db.Save(&Product{Code: "dont_save", Price: 100}).Error == nil {
		t.Errorf("An error from create callbacks expected when create")
	}

	p2 := Product{Code: "update_callback", Price: 100}
	db.Save(&p2)
	p2.Code = "dont_update"
	if db.Save(&p2).Error == nil {
		t.Errorf("An error from callbacks expected when update")
	}
	if db.Where("code = ?", "update_callback").First(&Product{}).Error != nil {
		t.Errorf("Record Should not be updated due to errors happened in callback")
	}
	if db.Where("code = ?", "dont_update").First(&Product{}).Error == nil {
		t.Errorf("Record Should not be updated due to errors happened in callback")
	}

	p2.Code = "dont_save"
	if db.Save(&p2).Error == nil {
		t.Errorf("An error from before save callbacks expected when update")
	}

	p3 := Product{Code: "dont_delete", Price: 100}
	db.Save(&p3)
	if db.Delete(&p3).Error == nil {
		t.Errorf("An error from before delete callbacks expected when delete")
	}

	if db.Where("Code = ?", "dont_delete").First(&p3).Error != nil {
		t.Errorf("Should not delete record due to errors happened in callback")
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

func TestNoUnExpectedHappenWithInvalidSql(t *testing.T) {
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

	db.Where("unexisting = ?", "3").Find(&[]User{})
}

func TestSetTableDirectly(t *testing.T) {
	var ages []int64
	if db.Table("users").Pluck("age", &ages).Error != nil {
		t.Errorf("No errors should happen if only set table")
	}

	if len(ages) == 0 {
		t.Errorf("Should find out some records")
	}

	var users []User
	if db.Table("users").Find(&users).Error != nil {
		t.Errorf("No errors should happen if set table to an existing table")
	}

	if db.Table("unexisting_users_table").Find(&users).Error == nil {
		t.Errorf("Should got some errors if set table to an unexisting table")
	}

	db.Exec("drop table deleted_users;")
	if db.Table("deleted_users").CreateTable(&User{}).Error != nil {
		t.Errorf("Should create deleted_users table")
	}

	db.Table("deleted_users").Save(&User{Name: "DeletedUser"})

	var deleted_users []User
	db.Table("deleted_users").Find(&deleted_users)
	if len(deleted_users) != 1 {
		t.Errorf("Should query from deleted_users table")
	}

	var deleted_user User
	db.Table("deleted_users").Find(&deleted_user)
	if deleted_user.Name != "DeletedUser" {
		t.Errorf("Should query from deleted_users table")
	}

	var user1, user2, user3 User
	db.First(&user1).Table("deleted_users").First(&user2).Table("").First(&user3)
	if !((user1.Name != user2.Name) && (user1.Name == user3.Name)) {
		t.Errorf("Set Table Chain Should works well")
	}
}

func TestUpdate(t *testing.T) {
	product1 := Product{Code: "123"}
	product2 := Product{Code: "234"}
	db.Save(&product1).Save(&product2).Update("code", "456")

	if db.First(&Product{}, "code = '123'").Error != nil {
		t.Errorf("Product 123's code should not be changed!")
	}

	if db.First(&Product{}, "code = '234'").Error == nil {
		t.Errorf("Product 234's code should be changed to 456!")
	}

	if db.First(&Product{}, "code = '456'").Error != nil {
		t.Errorf("Product 234's code should be 456 now!")
	}

	db.Table("products").Where("code in (?)", []string{"123"}).Update("code", "789")
	if db.First(&Product{}, "code = '123'").Error == nil {
		t.Errorf("Product 123's code should be changed to 789")
	}

	if db.First(&Product{}, "code = '456'").Error != nil {
		t.Errorf("Product 456's code should not be changed to 789")
	}

	if db.First(&Product{}, "code = '789'").Error != nil {
		t.Errorf("We should have Product 789")
	}
}

func TestUpdates(t *testing.T) {
	product1 := Product{Code: "abc", Price: 10}
	product2 := Product{Code: "cde", Price: 20}
	db.Save(&product1).Save(&product2).Updates(map[string]interface{}{"code": "edf", "price": 100})

	if db.First(&Product{}, "code = 'abc' and price = 10").Error != nil {
		t.Errorf("Product abc should not be updated!")
	}

	if db.First(&Product{}, "code = 'cde'").Error == nil {
		t.Errorf("Product cde should be renamed to edf!")
	}

	if db.First(&Product{}, "code = 'edf' and price = 100").Error != nil {
		t.Errorf("We should have product edf!")
	}

	db.Table("products").Where("code in (?)", []string{"abc"}).Updates(map[string]interface{}{"code": "fgh", "price": 200})
	if db.First(&Product{}, "code = 'abc'").Error == nil {
		t.Errorf("Product abc's code should be changed to fgh")
	}

	if db.First(&Product{}, "code = 'edf' and price = ?", 100).Error != nil {
		t.Errorf("Product cde's code should not be changed to fgh")
	}

	if db.First(&Product{}, "code = 'fgh' and price = 200").Error != nil {
		t.Errorf("We should have Product fgh")
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
		t.Errorf("Can't find the user because it is soft deleted")
	}

	if db.Unscoped().First(&Order{}, "amount = 1234").Error != nil {
		t.Errorf("Should be able to find out the soft deleted user with unscoped")
	}

	db.Unscoped().Delete(&order)
	if db.Unscoped().First(&Order{}, "amount = 1234").Error == nil {
		t.Errorf("Can't find out permanently deleted order")
	}
}

func TestFindOrInitialize(t *testing.T) {
	var user1 User
	db.Where(&User{Name: "hello world", Age: 33}).FirstOrInit(&user1)
	if user1.Name != "hello world" || user1.Id != 0 || user1.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	// db.FirstOrInit(&user2, map[string]interface{}{"name": "hahaha"})
	// if user2.Name != "hahaha" || user2.Id != 0 {
	// 	t.Errorf("user should be initialized with search value")
	// }
}
