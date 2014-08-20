package gorm_test

import (
	"fmt"
	"github.com/jinzhu/now"
	"reflect"

	"testing"
	"time"
)

func TestFirstAndLast(t *testing.T) {
	db.Save(&User{Name: "user1", Emails: []Email{{Email: "user1@example.com"}}})
	db.Save(&User{Name: "user2", Emails: []Email{{Email: "user2@example.com"}}})

	var user1, user2, user3, user4 User
	db.First(&user1)
	db.Order("id").Find(&user2)

	db.Last(&user3)
	db.Order("id desc").Find(&user4)
	if user1.Id != user2.Id || user3.Id != user4.Id {
		t.Errorf("First and Last should by order by primary key")
	}

	var users []User
	db.First(&users)
	if len(users) != 1 {
		t.Errorf("Find first record as slice")
	}

	if db.Joins("left join emails on emails.user_id = users.id").First(&User{}).Error != nil {
		t.Errorf("Should not raise any error when order with Join table")
	}
}

func TestFirstAndLastWithNoStdPrimaryKey(t *testing.T) {
	db.Save(&Animal{Name: "animal1"})
	db.Save(&Animal{Name: "animal2"})

	var animal1, animal2, animal3, animal4 Animal
	db.First(&animal1)
	db.Order("counter").Find(&animal2)

	db.Last(&animal3)
	db.Order("counter desc").Find(&animal4)
	if animal1.Counter != animal2.Counter || animal3.Counter != animal4.Counter {
		t.Errorf("First and Last should work correctly")
	}
}

func TestFindAsSliceOfPointers(t *testing.T) {
	db.Save(&User{Name: "user"})

	var users []User
	db.Find(&users)

	var userPointers []*User
	db.Find(&userPointers)

	if len(users) == 0 || len(users) != len(userPointers) {
		t.Errorf("Find slice of pointers")
	}
}

func TestSearchWithPlainSQL(t *testing.T) {
	user1 := User{Name: "PlainSqlUser1", Age: 1, Birthday: now.MustParse("2000-1-1")}
	user2 := User{Name: "PlainSqlUser2", Age: 10, Birthday: now.MustParse("2010-1-1")}
	user3 := User{Name: "PlainSqlUser3", Age: 20, Birthday: now.MustParse("2020-1-1")}
	db.Save(&user1).Save(&user2).Save(&user3)
	scopedb := db.Where("name LIKE ?", "%PlainSqlUser%")

	if db.Where("name = ?", user1.Name).First(&User{}).RecordNotFound() {
		t.Errorf("Search with plain SQL")
	}

	if db.Where("name LIKE ?", "%"+user1.Name+"%").First(&User{}).RecordNotFound() {
		t.Errorf("Search with plan SQL (regexp)")
	}

	var users []User
	db.Find(&users, "name LIKE ? and age > ?", "%PlainSqlUser%", 1)
	if len(users) != 2 {
		t.Errorf("Should found 2 users that age > 1, but got %v", len(users))
	}

	users = []User{}
	db.Where("name LIKE ?", "%PlainSqlUser%").Where("age >= ?", 1).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users that age >= 1, but got %v", len(users))
	}

	users = []User{}
	scopedb.Where("age <> ?", 20).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users age != 20, but got %v", len(users))
	}

	users = []User{}
	scopedb.Where("birthday > ?", now.MustParse("2000-1-1")).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users's birthday > 2000-1-1, but got %v", len(users))
	}

	users = []User{}
	scopedb.Where("birthday > ?", "2002-10-10").Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users's birthday >= 2002-10-10, but got %v", len(users))
	}

	users = []User{}
	scopedb.Where("birthday >= ?", "2010-1-1").Where("birthday < ?", "2020-1-1").Find(&users)
	if len(users) != 1 {
		t.Errorf("Should found 1 users's birthday < 2020-1-1 and >= 2010-1-1, but got %v", len(users))
	}

	users = []User{}
	db.Where("name in (?)", []string{user1.Name, user2.Name}).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users, but got %v", len(users))
	}

	users = []User{}
	db.Where("id in (?)", []int64{user1.Id, user2.Id, user3.Id}).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users, but got %v", len(users))
	}

	users = []User{}
	db.Where("id in (?)", user1.Id).Find(&users)
	if len(users) != 1 {
		t.Errorf("Should found 1 users, but got %v", len(users))
	}

	if !db.Where("name = ?", "none existing").Find(&[]User{}).RecordNotFound() {
		t.Errorf("Should get RecordNotFound error when looking for none existing records")
	}
}

func TestSearchWithStruct(t *testing.T) {
	user1 := User{Name: "StructSearchUser1", Age: 1, Birthday: now.MustParse("2000-1-1")}
	user2 := User{Name: "StructSearchUser2", Age: 10, Birthday: now.MustParse("2010-1-1")}
	user3 := User{Name: "StructSearchUser3", Age: 20, Birthday: now.MustParse("2020-1-1")}
	db.Save(&user1).Save(&user2).Save(&user3)

	if db.Where(user1.Id).First(&User{}).RecordNotFound() {
		t.Errorf("Search with primary key")
	}

	if db.First(&User{}, user1.Id).RecordNotFound() {
		t.Errorf("Search with primary key as inline condition")
	}

	if db.First(&User{}, fmt.Sprintf("%v", user1.Id)).RecordNotFound() {
		t.Errorf("Search with primary key as inline condition")
	}

	var users []User
	db.Where([]int64{user1.Id, user2.Id, user3.Id}).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users when search with primary keys, but got %v", len(users))
	}

	var user User
	db.First(&user, &User{Name: user1.Name})
	if user.Id == 0 || user.Name != user1.Name {
		t.Errorf("Search first record with inline pointer of struct")
	}

	db.First(&user, User{Name: user1.Name})
	if user.Id == 0 || user.Name != user.Name {
		t.Errorf("Search first record with inline struct")
	}

	db.Where(&User{Name: user1.Name}).First(&user)
	if user.Id == 0 || user.Name != user1.Name {
		t.Errorf("Search first record with where struct")
	}

	users = []User{}
	db.Find(&users, &User{Name: user2.Name})
	if len(users) != 1 {
		t.Errorf("Search all records with inline struct")
	}
}

func TestSearchWithMap(t *testing.T) {
	user1 := User{Name: "MapSearchUser1", Age: 1, Birthday: now.MustParse("2000-1-1")}
	user2 := User{Name: "MapSearchUser2", Age: 10, Birthday: now.MustParse("2010-1-1")}
	user3 := User{Name: "MapSearchUser3", Age: 20, Birthday: now.MustParse("2020-1-1")}
	db.Save(&user1).Save(&user2).Save(&user3)

	var user User
	db.First(&user, map[string]interface{}{"name": user1.Name})
	if user.Id == 0 || user.Name != user1.Name {
		t.Errorf("Search first record with inline map")
	}

	user = User{}
	db.Where(map[string]interface{}{"name": user2.Name}).First(&user)
	if user.Id == 0 || user.Name != user2.Name {
		t.Errorf("Search first record with where map")
	}

	var users []User
	db.Where(map[string]interface{}{"name": user3.Name}).Find(&users)
	if len(users) != 1 {
		t.Errorf("Search all records with inline map")
	}

	users = []User{}
	db.Find(&users, map[string]interface{}{"name": user3.Name})
	if len(users) != 1 {
		t.Errorf("Search all records with inline map")
	}
}

func TestSelect(t *testing.T) {
	user1 := User{Name: "SelectUser1"}
	db.Save(&user1)

	var user User
	db.Where("name = ?", user1.Name).Select("name").Find(&user)
	if user.Id != 0 {
		t.Errorf("Should not have ID because only selected name, %+v", user.Id)
	}

	if user.Name != user1.Name {
		t.Errorf("Should have user Name when selected it")
	}
}

func TestOrderAndPluck(t *testing.T) {
	user1 := User{Name: "OrderPluckUser1", Age: 1}
	user2 := User{Name: "OrderPluckUser2", Age: 10}
	user3 := User{Name: "OrderPluckUser3", Age: 20}
	db.Save(&user1).Save(&user2).Save(&user3)
	scopedb := db.Model(&User{}).Where("name like ?", "%OrderPluckUser%")

	var ages []int64
	scopedb.Order("age desc").Pluck("age", &ages)
	if ages[0] != 20 {
		t.Errorf("The first age should be 20 when order with age desc")
	}

	var ages1, ages2 []int64
	scopedb.Order("age desc").Pluck("age", &ages1).Order("age").Pluck("age", &ages2)
	if !reflect.DeepEqual(ages1, ages2) {
		t.Errorf("The first order is the primary order")
	}

	var ages3, ages4 []int64
	scopedb.Model(&User{}).Order("age desc").Pluck("age", &ages3).Order("age", true).Pluck("age", &ages4)
	if reflect.DeepEqual(ages3, ages4) {
		t.Errorf("Reorder should work")
	}

	var names []string
	var ages5 []int64
	scopedb.Model(User{}).Order("name").Order("age desc").Pluck("age", &ages5).Pluck("name", &names)
	if !(names[0] == user1.Name && names[1] == user2.Name && names[2] == user3.Name && ages5[2] == 20) {
		t.Errorf("Order with multiple orders")
	}

	db.Model(User{}).Select("name, age").Find(&[]User{})
}

func TestLimit(t *testing.T) {
	user1 := User{Name: "LimitUser1", Age: 1}
	user2 := User{Name: "LimitUser2", Age: 10}
	user3 := User{Name: "LimitUser3", Age: 20}
	user4 := User{Name: "LimitUser4", Age: 10}
	user5 := User{Name: "LimitUser5", Age: 20}
	db.Save(&user1).Save(&user2).Save(&user3).Save(&user4).Save(&user5)

	var users1, users2, users3 []User
	db.Order("age desc").Limit(3).Find(&users1).Limit(5).Find(&users2).Limit(-1).Find(&users3)

	if len(users1) != 3 || len(users2) != 5 || len(users3) <= 5 {
		t.Errorf("Limit should works")
	}
}

func TestOffset(t *testing.T) {
	for i := 0; i < 20; i++ {
		db.Save(&User{Name: fmt.Sprintf("OffsetUser%v", i)})
	}
	var users1, users2, users3, users4 []User
	db.Limit(100).Order("age desc").Find(&users1).Offset(3).Find(&users2).Offset(5).Find(&users3).Offset(-1).Find(&users4)

	if (len(users1) != len(users4)) || (len(users1)-len(users2) != 3) || (len(users1)-len(users3) != 5) {
		t.Errorf("Offset should work")
	}
}

func TestOr(t *testing.T) {
	user1 := User{Name: "OrUser1", Age: 1}
	user2 := User{Name: "OrUser2", Age: 10}
	user3 := User{Name: "OrUser3", Age: 20}
	db.Save(&user1).Save(&user2).Save(&user3)

	var users []User
	db.Where("name = ?", user1.Name).Or("name = ?", user2.Name).Find(&users)
	if len(users) != 2 {
		t.Errorf("Find users with or")
	}
}

func TestCount(t *testing.T) {
	user1 := User{Name: "CountUser1", Age: 1}
	user2 := User{Name: "CountUser2", Age: 10}
	user3 := User{Name: "CountUser3", Age: 20}

	db.Save(&user1).Save(&user2).Save(&user3)
	var count, count1, count2 int64
	var users []User

	if err := db.Where("name = ?", user1.Name).Or("name = ?", user3.Name).Find(&users).Count(&count).Error; err != nil {
		t.Errorf(fmt.Sprintf("Count should work, but got err %v", err))
	}

	if count != int64(len(users)) {
		t.Errorf("Count() method should get correct value")
	}

	db.Model(&User{}).Where("name = ?", user1.Name).Count(&count1).Or("name in (?)", []string{user2.Name, user3.Name}).Count(&count2)
	if count1 != 1 || count2 != 3 {
		t.Errorf("Multiple count in chain")
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

	var name3Count int64
	db.Table("users").Where("name = ?", "3").Count(&name3Count)
	db.Not("name", "3").Find(&users4)
	if len(users1)-len(users4) != int(name3Count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	users4 = []User{}
	db.Not("name = ?", "3").Find(&users4)
	if len(users1)-len(users4) != int(name3Count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	users4 = []User{}
	db.Not("name <> ?", "3").Find(&users4)
	if len(users4) != int(name3Count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	db.Not(User{Name: "3"}).Find(&users5)

	if len(users1)-len(users5) != int(name3Count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	db.Not(map[string]interface{}{"name": "3"}).Find(&users6)
	if len(users1)-len(users6) != int(name3Count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	db.Not("name", []string{"3"}).Find(&users7)
	if len(users1)-len(users7) != int(name3Count) {
		t.Errorf("Should find all users's name not equal 3")
	}

	var name2Count int64
	db.Table("users").Where("name = ?", "2").Count(&name2Count)
	db.Not("name", []string{"3", "2"}).Find(&users8)
	if len(users1)-len(users8) != (int(name3Count) + int(name2Count)) {
		t.Errorf("Should find all users's name not equal 3")
	}
}

func TestFillSmallerStruct(t *testing.T) {
	user1 := User{Name: "SmallerUser", Age: 100}
	db.Save(&user1)
	type SimpleUser struct {
		Name      string
		Id        int64
		UpdatedAt time.Time
		CreatedAt time.Time
	}

	var simpleUser SimpleUser
	db.Table("users").Where("name = ?", user1.Name).First(&simpleUser)

	if simpleUser.Id == 0 || simpleUser.Name == "" {
		t.Errorf("Should fill data correctly into smaller struct")
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

	updatedAt1 := user4.UpdatedAt
	db.Where(&User{Name: "find or create 3"}).Assign("age", 55).FirstOrCreate(&user4)
	if updatedAt1.Format(time.RFC3339Nano) == user4.UpdatedAt.Format(time.RFC3339Nano) {
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

func TestSelectWithEscapedFieldName(t *testing.T) {
	user1 := User{Name: "EscapedFieldNameUser", Age: 1}
	user2 := User{Name: "EscapedFieldNameUser", Age: 10}
	user3 := User{Name: "EscapedFieldNameUser", Age: 20}
	db.Save(&user1).Save(&user2).Save(&user3)

	var names []string
	db.Model(User{}).Where(&User{Name: "EscapedFieldNameUser"}).Pluck("\"name\"", &names)

	if len(names) != 3 {
		t.Errorf("Expected 3 name, but got: %d", len(names))
	}
}
