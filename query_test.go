package gorm_test

import (
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"

	"testing"
	"time"
)

func TestFirstAndLast(t *testing.T) {
	DB.Save(&User{Name: "user1", Emails: []Email{{Email: "user1@example.com"}}})
	DB.Save(&User{Name: "user2", Emails: []Email{{Email: "user2@example.com"}}})

	var user1, user2, user3, user4 User
	DB.First(&user1)
	DB.Order("id").Limit(1).Find(&user2)

	ptrOfUser3 := &user3
	DB.Last(&ptrOfUser3)
	DB.Order("id desc").Limit(1).Find(&user4)
	if user1.Id != user2.Id || user3.Id != user4.Id {
		t.Errorf("First and Last should by order by primary key")
	}

	var users []User
	DB.First(&users)
	if len(users) != 1 {
		t.Errorf("Find first record as slice")
	}

	var user User
	if DB.Joins("left join emails on emails.user_id = users.id").First(&user).Error != nil {
		t.Errorf("Should not raise any error when order with Join table")
	}

	if user.Email != "" {
		t.Errorf("User's Email should be blank as no one set it")
	}
}

func TestFirstAndLastWithNoStdPrimaryKey(t *testing.T) {
	DB.Save(&Animal{Name: "animal1"})
	DB.Save(&Animal{Name: "animal2"})

	var animal1, animal2, animal3, animal4 Animal
	DB.First(&animal1)
	DB.Order("counter").Limit(1).Find(&animal2)

	DB.Last(&animal3)
	DB.Order("counter desc").Limit(1).Find(&animal4)
	if animal1.Counter != animal2.Counter || animal3.Counter != animal4.Counter {
		t.Errorf("First and Last should work correctly")
	}
}

func TestFirstAndLastWithRaw(t *testing.T) {
	user1 := User{Name: "user", Emails: []Email{{Email: "user1@example.com"}}}
	user2 := User{Name: "user", Emails: []Email{{Email: "user2@example.com"}}}
	DB.Save(&user1)
	DB.Save(&user2)

	var user3, user4 User
	DB.Raw("select * from users WHERE name = ?", "user").First(&user3)
	if user3.Id != user1.Id {
		t.Errorf("Find first record with raw")
	}

	DB.Raw("select * from users WHERE name = ?", "user").Last(&user4)
	if user4.Id != user2.Id {
		t.Errorf("Find last record with raw")
	}
}

func TestUIntPrimaryKey(t *testing.T) {
	var animal Animal
	DB.First(&animal, uint64(1))
	if animal.Counter != 1 {
		t.Errorf("Fetch a record from with a non-int primary key should work, but failed")
	}

	DB.Model(Animal{}).Where(Animal{Counter: uint64(2)}).Scan(&animal)
	if animal.Counter != 2 {
		t.Errorf("Fetch a record from with a non-int primary key should work, but failed")
	}
}

func TestCustomizedTypePrimaryKey(t *testing.T) {
	type ID uint
	type CustomizedTypePrimaryKey struct {
		ID   ID
		Name string
	}

	DB.AutoMigrate(&CustomizedTypePrimaryKey{})

	p1 := CustomizedTypePrimaryKey{Name: "p1"}
	p2 := CustomizedTypePrimaryKey{Name: "p2"}
	p3 := CustomizedTypePrimaryKey{Name: "p3"}
	DB.Create(&p1)
	DB.Create(&p2)
	DB.Create(&p3)

	var p CustomizedTypePrimaryKey

	if err := DB.First(&p, p2.ID).Error; err == nil {
		t.Errorf("Should return error for invalid query condition")
	}

	if err := DB.First(&p, "id = ?", p2.ID).Error; err != nil {
		t.Errorf("No error should happen when querying with customized type for primary key, got err %v", err)
	}

	if p.Name != "p2" {
		t.Errorf("Should find correct value when querying with customized type for primary key")
	}
}

func TestStringPrimaryKeyForNumericValueStartingWithZero(t *testing.T) {
	type AddressByZipCode struct {
		ZipCode string `gorm:"primary_key"`
		Address string
	}

	DB.AutoMigrate(&AddressByZipCode{})
	DB.Create(&AddressByZipCode{ZipCode: "00501", Address: "Holtsville"})

	var address AddressByZipCode
	DB.First(&address, "00501")
	if address.ZipCode != "00501" {
		t.Errorf("Fetch a record from with a string primary key for a numeric value starting with zero should work, but failed, zip code is %v", address.ZipCode)
	}
}

func TestFindAsSliceOfPointers(t *testing.T) {
	DB.Save(&User{Name: "user"})

	var users []User
	DB.Find(&users)

	var userPointers []*User
	DB.Find(&userPointers)

	if len(users) == 0 || len(users) != len(userPointers) {
		t.Errorf("Find slice of pointers")
	}
}

func TestSearchWithPlainSQL(t *testing.T) {
	user1 := User{Name: "PlainSqlUser1", Age: 1, Birthday: parseTime("2000-1-1")}
	user2 := User{Name: "PlainSqlUser2", Age: 10, Birthday: parseTime("2010-1-1")}
	user3 := User{Name: "PlainSqlUser3", Age: 20, Birthday: parseTime("2020-1-1")}
	DB.Save(&user1).Save(&user2).Save(&user3)
	scopedb := DB.Where("name LIKE ?", "%PlainSqlUser%")

	if DB.Where("name = ?", user1.Name).First(&User{}).RecordNotFound() {
		t.Errorf("Search with plain SQL")
	}

	if DB.Where("name LIKE ?", "%"+user1.Name+"%").First(&User{}).RecordNotFound() {
		t.Errorf("Search with plan SQL (regexp)")
	}

	var users []User
	DB.Find(&users, "name LIKE ? and age > ?", "%PlainSqlUser%", 1)
	if len(users) != 2 {
		t.Errorf("Should found 2 users that age > 1, but got %v", len(users))
	}

	DB.Where("name LIKE ?", "%PlainSqlUser%").Where("age >= ?", 1).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users that age >= 1, but got %v", len(users))
	}

	scopedb.Where("age <> ?", 20).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users age != 20, but got %v", len(users))
	}

	scopedb.Where("birthday > ?", parseTime("2000-1-1")).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users' birthday > 2000-1-1, but got %v", len(users))
	}

	scopedb.Where("birthday > ?", "2002-10-10").Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users' birthday >= 2002-10-10, but got %v", len(users))
	}

	scopedb.Where("birthday >= ?", "2010-1-1").Where("birthday < ?", "2020-1-1").Find(&users)
	if len(users) != 1 {
		t.Errorf("Should found 1 users' birthday < 2020-1-1 and >= 2010-1-1, but got %v", len(users))
	}

	DB.Where("name in (?)", []string{user1.Name, user2.Name}).Find(&users)
	if len(users) != 2 {
		t.Errorf("Should found 2 users, but got %v", len(users))
	}

	DB.Where("id in (?)", []int64{user1.Id, user2.Id, user3.Id}).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users, but got %v", len(users))
	}

	DB.Where("id in (?)", user1.Id).Find(&users)
	if len(users) != 1 {
		t.Errorf("Should found 1 users, but got %v", len(users))
	}

	if err := DB.Where("id IN (?)", []string{}).Find(&users).Error; err != nil {
		t.Error("no error should happen when query with empty slice, but got: ", err)
	}

	if err := DB.Not("id IN (?)", []string{}).Find(&users).Error; err != nil {
		t.Error("no error should happen when query with empty slice, but got: ", err)
	}

	if DB.Where("name = ?", "none existing").Find(&[]User{}).RecordNotFound() {
		t.Errorf("Should not get RecordNotFound error when looking for none existing records")
	}
}

func TestSearchWithTwoDimensionalArray(t *testing.T) {
	var users []User
	user1 := User{Name: "2DSearchUser1", Age: 1, Birthday: parseTime("2000-1-1")}
	user2 := User{Name: "2DSearchUser2", Age: 10, Birthday: parseTime("2010-1-1")}
	user3 := User{Name: "2DSearchUser3", Age: 20, Birthday: parseTime("2020-1-1")}
	DB.Create(&user1)
	DB.Create(&user2)
	DB.Create(&user3)

	if dialect := DB.Dialect().GetName(); dialect == "mysql" || dialect == "postgres" {
		if err := DB.Where("(name, age) IN (?)", [][]interface{}{{"2DSearchUser1", 1}, {"2DSearchUser2", 10}}).Find(&users).Error; err != nil {
			t.Errorf("No error should happen when query with 2D array, but got %v", err)

			if len(users) != 2 {
				t.Errorf("Should find 2 users with 2D array, but got %v", len(users))
			}
		}
	}

	if dialect := DB.Dialect().GetName(); dialect == "mssql" {
		if err := DB.Joins("JOIN (VALUES ?) AS x (col1, col2) ON x.col1 = name AND x.col2 = age", [][]interface{}{{"2DSearchUser1", 1}, {"2DSearchUser2", 10}}).Find(&users).Error; err != nil {
			t.Errorf("No error should happen when query with 2D array, but got %v", err)

			if len(users) != 2 {
				t.Errorf("Should find 2 users with 2D array, but got %v", len(users))
			}
		}
	}
}

func TestSearchWithStruct(t *testing.T) {
	user1 := User{Name: "StructSearchUser1", Age: 1, Birthday: parseTime("2000-1-1")}
	user2 := User{Name: "StructSearchUser2", Age: 10, Birthday: parseTime("2010-1-1")}
	user3 := User{Name: "StructSearchUser3", Age: 20, Birthday: parseTime("2020-1-1")}
	DB.Save(&user1).Save(&user2).Save(&user3)

	if DB.Where(user1.Id).First(&User{}).RecordNotFound() {
		t.Errorf("Search with primary key")
	}

	if DB.First(&User{}, user1.Id).RecordNotFound() {
		t.Errorf("Search with primary key as inline condition")
	}

	if DB.First(&User{}, fmt.Sprintf("%v", user1.Id)).RecordNotFound() {
		t.Errorf("Search with primary key as inline condition")
	}

	var users []User
	DB.Where([]int64{user1.Id, user2.Id, user3.Id}).Find(&users)
	if len(users) != 3 {
		t.Errorf("Should found 3 users when search with primary keys, but got %v", len(users))
	}

	var user User
	DB.First(&user, &User{Name: user1.Name})
	if user.Id == 0 || user.Name != user1.Name {
		t.Errorf("Search first record with inline pointer of struct")
	}

	DB.First(&user, User{Name: user1.Name})
	if user.Id == 0 || user.Name != user1.Name {
		t.Errorf("Search first record with inline struct")
	}

	DB.Where(&User{Name: user1.Name}).First(&user)
	if user.Id == 0 || user.Name != user1.Name {
		t.Errorf("Search first record with where struct")
	}

	DB.Find(&users, &User{Name: user2.Name})
	if len(users) != 1 {
		t.Errorf("Search all records with inline struct")
	}
}

func TestSearchWithMap(t *testing.T) {
	companyID := 1
	user1 := User{Name: "MapSearchUser1", Age: 1, Birthday: parseTime("2000-1-1")}
	user2 := User{Name: "MapSearchUser2", Age: 10, Birthday: parseTime("2010-1-1")}
	user3 := User{Name: "MapSearchUser3", Age: 20, Birthday: parseTime("2020-1-1")}
	user4 := User{Name: "MapSearchUser4", Age: 30, Birthday: parseTime("2020-1-1"), CompanyID: &companyID}
	DB.Save(&user1).Save(&user2).Save(&user3).Save(&user4)

	var user User
	DB.First(&user, map[string]interface{}{"name": user1.Name})
	if user.Id == 0 || user.Name != user1.Name {
		t.Errorf("Search first record with inline map")
	}

	user = User{}
	DB.Where(map[string]interface{}{"name": user2.Name}).First(&user)
	if user.Id == 0 || user.Name != user2.Name {
		t.Errorf("Search first record with where map")
	}

	var users []User
	DB.Where(map[string]interface{}{"name": user3.Name}).Find(&users)
	if len(users) != 1 {
		t.Errorf("Search all records with inline map")
	}

	DB.Find(&users, map[string]interface{}{"name": user3.Name})
	if len(users) != 1 {
		t.Errorf("Search all records with inline map")
	}

	DB.Find(&users, map[string]interface{}{"name": user4.Name, "company_id": nil})
	if len(users) != 0 {
		t.Errorf("Search all records with inline map containing null value finding 0 records")
	}

	DB.Find(&users, map[string]interface{}{"name": user1.Name, "company_id": nil})
	if len(users) != 1 {
		t.Errorf("Search all records with inline map containing null value finding 1 record")
	}

	DB.Find(&users, map[string]interface{}{"name": user4.Name, "company_id": companyID})
	if len(users) != 1 {
		t.Errorf("Search all records with inline multiple value map")
	}
}

func TestSearchWithEmptyChain(t *testing.T) {
	user1 := User{Name: "ChainSearchUser1", Age: 1, Birthday: parseTime("2000-1-1")}
	user2 := User{Name: "ChainearchUser2", Age: 10, Birthday: parseTime("2010-1-1")}
	user3 := User{Name: "ChainearchUser3", Age: 20, Birthday: parseTime("2020-1-1")}
	DB.Save(&user1).Save(&user2).Save(&user3)

	if DB.Where("").Where("").First(&User{}).Error != nil {
		t.Errorf("Should not raise any error if searching with empty strings")
	}

	if DB.Where(&User{}).Where("name = ?", user1.Name).First(&User{}).Error != nil {
		t.Errorf("Should not raise any error if searching with empty struct")
	}

	if DB.Where(map[string]interface{}{}).Where("name = ?", user1.Name).First(&User{}).Error != nil {
		t.Errorf("Should not raise any error if searching with empty map")
	}
}

func TestSelect(t *testing.T) {
	user1 := User{Name: "SelectUser1"}
	DB.Save(&user1)

	var user User
	DB.Where("name = ?", user1.Name).Select("name").Find(&user)
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
	DB.Save(&user1).Save(&user2).Save(&user3)
	scopedb := DB.Model(&User{}).Where("name like ?", "%OrderPluckUser%")

	var user User
	scopedb.Order(gorm.Expr("case when name = ? then 0 else 1 end", "OrderPluckUser2")).First(&user)
	if user.Name != "OrderPluckUser2" {
		t.Errorf("Order with sql expression")
	}

	var ages []int64
	scopedb.Order("age desc").Pluck("age", &ages)
	if ages[0] != 20 {
		t.Errorf("The first age should be 20 when order with age desc")
	}

	var ages1, ages2 []int64
	scopedb.Order("age desc").Pluck("age", &ages1).Pluck("age", &ages2)
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
	if names != nil && ages5 != nil {
		if !(names[0] == user1.Name && names[1] == user2.Name && names[2] == user3.Name && ages5[2] == 20) {
			t.Errorf("Order with multiple orders")
		}
	} else {
		t.Errorf("Order with multiple orders")
	}

	var ages6 []int64
	if err := scopedb.Order("").Pluck("age", &ages6).Error; err != nil {
		t.Errorf("An empty string as order clause produces invalid queries")
	}

	DB.Model(User{}).Select("name, age").Find(&[]User{})
}

func TestLimit(t *testing.T) {
	user1 := User{Name: "LimitUser1", Age: 1}
	user2 := User{Name: "LimitUser2", Age: 10}
	user3 := User{Name: "LimitUser3", Age: 20}
	user4 := User{Name: "LimitUser4", Age: 10}
	user5 := User{Name: "LimitUser5", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3).Save(&user4).Save(&user5)

	var users1, users2, users3 []User
	DB.Order("age desc").Limit(3).Find(&users1).Limit(5).Find(&users2).Limit(-1).Find(&users3)

	if len(users1) != 3 || len(users2) != 5 || len(users3) <= 5 {
		t.Errorf("Limit should works")
	}
}

func TestOffset(t *testing.T) {
	for i := 0; i < 20; i++ {
		DB.Save(&User{Name: fmt.Sprintf("OffsetUser%v", i)})
	}
	var users1, users2, users3, users4 []User
	DB.Limit(100).Where("name like ?", "OffsetUser%").Order("age desc").Find(&users1).Offset(3).Find(&users2).Offset(5).Find(&users3).Offset(-1).Find(&users4)

	if (len(users1) != len(users4)) || (len(users1)-len(users2) != 3) || (len(users1)-len(users3) != 5) {
		t.Errorf("Offset should work")
	}
}

func TestLimitAndOffsetSQL(t *testing.T) {
	user1 := User{Name: "TestLimitAndOffsetSQL1", Age: 10}
	user2 := User{Name: "TestLimitAndOffsetSQL2", Age: 20}
	user3 := User{Name: "TestLimitAndOffsetSQL3", Age: 30}
	user4 := User{Name: "TestLimitAndOffsetSQL4", Age: 40}
	user5 := User{Name: "TestLimitAndOffsetSQL5", Age: 50}
	if err := DB.Save(&user1).Save(&user2).Save(&user3).Save(&user4).Save(&user5).Error; err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		limit, offset interface{}
		users         []*User
		ok            bool
	}{
		{
			name:   "OK",
			limit:  float64(2),
			offset: float64(2),
			users: []*User{
				&User{Name: "TestLimitAndOffsetSQL3", Age: 30},
				&User{Name: "TestLimitAndOffsetSQL2", Age: 20},
			},
			ok: true,
		},
		{
			name:   "Limit parse error",
			limit:  float64(1000000), // 1e+06
			offset: float64(2),
			ok:     false,
		},
		{
			name:   "Offset parse error",
			limit:  float64(2),
			offset: float64(1000000), // 1e+06
			ok:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var users []*User
			err := DB.Where("name LIKE ?", "TestLimitAndOffsetSQL%").Order("age desc").Limit(tt.limit).Offset(tt.offset).Find(&users).Error
			if tt.ok {
				if err != nil {
					t.Errorf("error expected nil, but got %v", err)
				}
				if len(users) != len(tt.users) {
					t.Errorf("users length expected %d, but got %d", len(tt.users), len(users))
				}
				for i := range tt.users {
					if users[i].Name != tt.users[i].Name {
						t.Errorf("users[%d] name expected %s, but got %s", i, tt.users[i].Name, users[i].Name)
					}
					if users[i].Age != tt.users[i].Age {
						t.Errorf("users[%d] age expected %d, but got %d", i, tt.users[i].Age, users[i].Age)
					}
				}
			} else {
				if err == nil {
					t.Error("error expected not nil, but got nil")
				}
			}
		})
	}
}

func TestOr(t *testing.T) {
	user1 := User{Name: "OrUser1", Age: 1}
	user2 := User{Name: "OrUser2", Age: 10}
	user3 := User{Name: "OrUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	var users []User
	DB.Where("name = ?", user1.Name).Or("name = ?", user2.Name).Find(&users)
	if len(users) != 2 {
		t.Errorf("Find users with or")
	}
}

func TestCount(t *testing.T) {
	user1 := User{Name: "CountUser1", Age: 1}
	user2 := User{Name: "CountUser2", Age: 10}
	user3 := User{Name: "CountUser3", Age: 20}

	DB.Save(&user1).Save(&user2).Save(&user3)
	var count, count1, count2 int64
	var users []User

	if err := DB.Where("name = ?", user1.Name).Or("name = ?", user3.Name).Find(&users).Count(&count).Error; err != nil {
		t.Errorf(fmt.Sprintf("Count should work, but got err %v", err))
	}

	if count != int64(len(users)) {
		t.Errorf("Count() method should get correct value")
	}

	DB.Model(&User{}).Where("name = ?", user1.Name).Count(&count1).Or("name in (?)", []string{user2.Name, user3.Name}).Count(&count2)
	if count1 != 1 || count2 != 3 {
		t.Errorf("Multiple count in chain")
	}

	var count3 int
	if err := DB.Model(&User{}).Where("name in (?)", []string{user2.Name, user2.Name, user3.Name}).Group("id").Count(&count3).Error; err != nil {
		t.Errorf("Not error should happen, but got %v", err)
	}

	if count3 != 2 {
		t.Errorf("Should get correct count, but got %v", count3)
	}
}

func TestNot(t *testing.T) {
	DB.Create(getPreparedUser("user1", "not"))
	DB.Create(getPreparedUser("user2", "not"))
	DB.Create(getPreparedUser("user3", "not"))

	user4 := getPreparedUser("user4", "not")
	user4.Company = Company{}
	DB.Create(user4)

	DB := DB.Where("role = ?", "not")

	var users1, users2, users3, users4, users5, users6, users7, users8, users9 []User
	if DB.Find(&users1).RowsAffected != 4 {
		t.Errorf("should find 4 not users")
	}
	DB.Not(users1[0].Id).Find(&users2)

	if len(users1)-len(users2) != 1 {
		t.Errorf("Should ignore the first users with Not")
	}

	DB.Not([]int{}).Find(&users3)
	if len(users1)-len(users3) != 0 {
		t.Errorf("Should find all users with a blank condition")
	}

	var name3Count int64
	DB.Table("users").Where("name = ?", "user3").Count(&name3Count)
	DB.Not("name", "user3").Find(&users4)
	if len(users1)-len(users4) != int(name3Count) {
		t.Errorf("Should find all users' name not equal 3")
	}

	DB.Not("name = ?", "user3").Find(&users4)
	if len(users1)-len(users4) != int(name3Count) {
		t.Errorf("Should find all users' name not equal 3")
	}

	DB.Not("name <> ?", "user3").Find(&users4)
	if len(users4) != int(name3Count) {
		t.Errorf("Should find all users' name not equal 3")
	}

	DB.Not(User{Name: "user3"}).Find(&users5)

	if len(users1)-len(users5) != int(name3Count) {
		t.Errorf("Should find all users' name not equal 3")
	}

	DB.Not(map[string]interface{}{"name": "user3"}).Find(&users6)
	if len(users1)-len(users6) != int(name3Count) {
		t.Errorf("Should find all users' name not equal 3")
	}

	DB.Not(map[string]interface{}{"name": "user3", "company_id": nil}).Find(&users7)
	if len(users1)-len(users7) != 2 { // not user3 or user4
		t.Errorf("Should find all user's name not equal to 3 who do not have company id")
	}

	DB.Not("name", []string{"user3"}).Find(&users8)
	if len(users1)-len(users8) != int(name3Count) {
		t.Errorf("Should find all users' name not equal 3")
	}

	var name2Count int64
	DB.Table("users").Where("name = ?", "user2").Count(&name2Count)
	DB.Not("name", []string{"user3", "user2"}).Find(&users9)
	if len(users1)-len(users9) != (int(name3Count) + int(name2Count)) {
		t.Errorf("Should find all users' name not equal 3")
	}
}

func TestFillSmallerStruct(t *testing.T) {
	user1 := User{Name: "SmallerUser", Age: 100}
	DB.Save(&user1)
	type SimpleUser struct {
		Name      string
		Id        int64
		UpdatedAt time.Time
		CreatedAt time.Time
	}

	var simpleUser SimpleUser
	DB.Table("users").Where("name = ?", user1.Name).First(&simpleUser)

	if simpleUser.Id == 0 || simpleUser.Name == "" {
		t.Errorf("Should fill data correctly into smaller struct")
	}
}

func TestFindOrInitialize(t *testing.T) {
	var user1, user2, user3, user4, user5, user6 User
	DB.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user1)
	if user1.Name != "find or init" || user1.Id != 0 || user1.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	DB.Where(User{Name: "find or init", Age: 33}).FirstOrInit(&user2)
	if user2.Name != "find or init" || user2.Id != 0 || user2.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	DB.FirstOrInit(&user3, map[string]interface{}{"name": "find or init 2"})
	if user3.Name != "find or init 2" || user3.Id != 0 {
		t.Errorf("user should be initialized with inline search value")
	}

	DB.Where(&User{Name: "find or init"}).Attrs(User{Age: 44}).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.Id != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and attrs")
	}

	DB.Where(&User{Name: "find or init"}).Assign("age", 44).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.Id != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and assign attrs")
	}

	DB.Save(&User{Name: "find or init", Age: 33})
	DB.Where(&User{Name: "find or init"}).Attrs("age", 44).FirstOrInit(&user5)
	if user5.Name != "find or init" || user5.Id == 0 || user5.Age != 33 {
		t.Errorf("user should be found and not initialized by Attrs")
	}

	DB.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.Id == 0 || user6.Age != 33 {
		t.Errorf("user should be found with FirstOrInit")
	}

	DB.Where(&User{Name: "find or init"}).Assign(User{Age: 44}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.Id == 0 || user6.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}
}

func TestFindOrCreate(t *testing.T) {
	var user1, user2, user3, user4, user5, user6, user7, user8 User
	DB.Where(&User{Name: "find or create", Age: 33}).FirstOrCreate(&user1)
	if user1.Name != "find or create" || user1.Id == 0 || user1.Age != 33 {
		t.Errorf("user should be created with search value")
	}

	DB.Where(&User{Name: "find or create", Age: 33}).FirstOrCreate(&user2)
	if user1.Id != user2.Id || user2.Name != "find or create" || user2.Id == 0 || user2.Age != 33 {
		t.Errorf("user should be created with search value")
	}

	DB.FirstOrCreate(&user3, map[string]interface{}{"name": "find or create 2"})
	if user3.Name != "find or create 2" || user3.Id == 0 {
		t.Errorf("user should be created with inline search value")
	}

	DB.Where(&User{Name: "find or create 3"}).Attrs("age", 44).FirstOrCreate(&user4)
	if user4.Name != "find or create 3" || user4.Id == 0 || user4.Age != 44 {
		t.Errorf("user should be created with search value and attrs")
	}

	updatedAt1 := user4.UpdatedAt
	DB.Where(&User{Name: "find or create 3"}).Assign("age", 55).FirstOrCreate(&user4)
	if updatedAt1.Format(time.RFC3339Nano) == user4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("UpdateAt should be changed when update values with assign")
	}

	DB.Where(&User{Name: "find or create 4"}).Assign(User{Age: 44}).FirstOrCreate(&user4)
	if user4.Name != "find or create 4" || user4.Id == 0 || user4.Age != 44 {
		t.Errorf("user should be created with search value and assigned attrs")
	}

	DB.Where(&User{Name: "find or create"}).Attrs("age", 44).FirstOrInit(&user5)
	if user5.Name != "find or create" || user5.Id == 0 || user5.Age != 33 {
		t.Errorf("user should be found and not initialized by Attrs")
	}

	DB.Where(&User{Name: "find or create"}).Assign(User{Age: 44}).FirstOrCreate(&user6)
	if user6.Name != "find or create" || user6.Id == 0 || user6.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}

	DB.Where(&User{Name: "find or create"}).Find(&user7)
	if user7.Name != "find or create" || user7.Id == 0 || user7.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}

	DB.Where(&User{Name: "find or create embedded struct"}).Assign(User{Age: 44, CreditCard: CreditCard{Number: "1231231231"}, Emails: []Email{{Email: "jinzhu@assign_embedded_struct.com"}, {Email: "jinzhu-2@assign_embedded_struct.com"}}}).FirstOrCreate(&user8)
	if DB.Where("email = ?", "jinzhu-2@assign_embedded_struct.com").First(&Email{}).RecordNotFound() {
		t.Errorf("embedded struct email should be saved")
	}

	if DB.Where("email = ?", "1231231231").First(&CreditCard{}).RecordNotFound() {
		t.Errorf("embedded struct credit card should be saved")
	}
}

func TestSelectWithEscapedFieldName(t *testing.T) {
	user1 := User{Name: "EscapedFieldNameUser", Age: 1}
	user2 := User{Name: "EscapedFieldNameUser", Age: 10}
	user3 := User{Name: "EscapedFieldNameUser", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	var names []string
	DB.Model(User{}).Where(&User{Name: "EscapedFieldNameUser"}).Pluck("\"name\"", &names)

	if len(names) != 3 {
		t.Errorf("Expected 3 name, but got: %d", len(names))
	}
}

func TestSelectWithVariables(t *testing.T) {
	DB.Save(&User{Name: "jinzhu"})

	rows, _ := DB.Table("users").Select("? as fake", gorm.Expr("name")).Rows()

	if !rows.Next() {
		t.Errorf("Should have returned at least one row")
	} else {
		columns, _ := rows.Columns()
		if !reflect.DeepEqual(columns, []string{"fake"}) {
			t.Errorf("Should only contains one column")
		}
	}

	rows.Close()
}

func TestSelectWithArrayInput(t *testing.T) {
	DB.Save(&User{Name: "jinzhu", Age: 42})

	var user User
	DB.Select([]string{"name", "age"}).Where("age = 42 AND name = 'jinzhu'").First(&user)

	if user.Name != "jinzhu" || user.Age != 42 {
		t.Errorf("Should have selected both age and name")
	}
}

func TestPluckWithSelect(t *testing.T) {
	var (
		user              = User{Name: "matematik7_pluck_with_select", Age: 25}
		combinedName      = fmt.Sprintf("%v%v", user.Name, user.Age)
		combineUserAgeSQL = fmt.Sprintf("concat(%v, %v)", DB.Dialect().Quote("name"), DB.Dialect().Quote("age"))
	)

	if dialect := DB.Dialect().GetName(); dialect == "sqlite3" {
		combineUserAgeSQL = fmt.Sprintf("(%v || %v)", DB.Dialect().Quote("name"), DB.Dialect().Quote("age"))
	}

	DB.Save(&user)

	selectStr := combineUserAgeSQL + " as user_age"
	var userAges []string
	err := DB.Model(&User{}).Where("age = ?", 25).Select(selectStr).Pluck("user_age", &userAges).Error
	if err != nil {
		t.Error(err)
	}

	if len(userAges) != 1 || userAges[0] != combinedName {
		t.Errorf("Should correctly pluck with select, got: %s", userAges)
	}

	selectStr = combineUserAgeSQL + fmt.Sprintf(" as %v", DB.Dialect().Quote("user_age"))
	userAges = userAges[:0]
	err = DB.Model(&User{}).Where("age = ?", 25).Select(selectStr).Pluck("user_age", &userAges).Error
	if err != nil {
		t.Error(err)
	}

	if len(userAges) != 1 || userAges[0] != combinedName {
		t.Errorf("Should correctly pluck with select, got: %s", userAges)
	}
}
