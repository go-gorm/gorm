package gorm

import (
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

var (
	db                 DB
	t1, t2, t3, t4, t5 time.Time
)

func init() {
	db, _ = Open("postgres", "user=gorm dbname=gorm sslmode=disable")
	db.Exec("drop table users;")

	orm := db.CreateTable(&User{})
	if orm.Error != nil {
		panic("No error should raise when create table")
	}

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
	db.First(user)
	if user.Name != name {
		t.Errorf("User should be saved and fetched correctly")
	}

	users := []User{}
	db.Find(&users)
}

func TestUpdate(t *testing.T) {
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
	db.Where("Name = ?", name).First(user)
	if user.Name != name {
		t.Errorf("Should found out user with name '%v'", name)
	}

	user = &User{}
	orm := db.Where("Name = ?", "noexisting-user").First(user)
	if orm.Error == nil {
		t.Errorf("Should return error when looking for none existing record, %+v", user)
	}

	users := []User{}
	orm = db.Where("Name = ?", "none-noexisting").Find(&users)
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

	var user_ids []int64
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

func TestOrAndNot(t *testing.T) {
	var users []User
	db.Where("name = ?", "1").Or("name = ?", "3").Find(&users)
	if len(users) != 3 {
		t.Errorf("Should find three users with name 1 and 3")
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
