package tests_test

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestFind(t *testing.T) {
	var users = []User{
		*GetUser("find", Config{}),
		*GetUser("find", Config{}),
		*GetUser("find", Config{}),
	}

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create users: %v", err)
	}

	t.Run("First", func(t *testing.T) {
		var first User
		if err := DB.Where("name = ?", "find").First(&first).Error; err != nil {
			t.Errorf("errors happened when query first: %v", err)
		} else {
			CheckUser(t, first, users[0])
		}
	})

	t.Run("Last", func(t *testing.T) {
		var last User
		if err := DB.Where("name = ?", "find").Last(&last).Error; err != nil {
			t.Errorf("errors happened when query last: %v", err)
		} else {
			CheckUser(t, last, users[2])
		}
	})

	var all []User
	if err := DB.Where("name = ?", "find").Find(&all).Error; err != nil || len(all) != 3 {
		t.Errorf("errors happened when query find: %v, length: %v", err, len(all))
	} else {
		for idx, user := range users {
			t.Run("FindAll#"+strconv.Itoa(idx+1), func(t *testing.T) {
				CheckUser(t, all[idx], user)
			})
		}
	}

	t.Run("FirstMap", func(t *testing.T) {
		var first = map[string]interface{}{}
		if err := DB.Model(&User{}).Where("name = ?", "find").First(first).Error; err != nil {
			t.Errorf("errors happened when query first: %v", err)
		} else {
			for _, name := range []string{"Name", "Age", "Birthday"} {
				t.Run(name, func(t *testing.T) {
					dbName := DB.NamingStrategy.ColumnName("", name)
					reflectValue := reflect.Indirect(reflect.ValueOf(users[0]))
					AssertEqual(t, first[dbName], reflectValue.FieldByName(name).Interface())
				})
			}
		}
	})

	t.Run("FirstPtrMap", func(t *testing.T) {
		var first = map[string]interface{}{}
		if err := DB.Model(&User{}).Where("name = ?", "find").First(&first).Error; err != nil {
			t.Errorf("errors happened when query first: %v", err)
		} else {
			for _, name := range []string{"Name", "Age", "Birthday"} {
				t.Run(name, func(t *testing.T) {
					dbName := DB.NamingStrategy.ColumnName("", name)
					reflectValue := reflect.Indirect(reflect.ValueOf(users[0]))
					AssertEqual(t, first[dbName], reflectValue.FieldByName(name).Interface())
				})
			}
		}
	})

	t.Run("FirstSliceOfMap", func(t *testing.T) {
		var allMap = []map[string]interface{}{}
		if err := DB.Model(&User{}).Where("name = ?", "find").Find(&allMap).Error; err != nil {
			t.Errorf("errors happened when query first: %v", err)
		} else {
			for idx, user := range users {
				t.Run("FindAllMap#"+strconv.Itoa(idx+1), func(t *testing.T) {
					for _, name := range []string{"Name", "Age", "Birthday"} {
						t.Run(name, func(t *testing.T) {
							dbName := DB.NamingStrategy.ColumnName("", name)
							reflectValue := reflect.Indirect(reflect.ValueOf(user))
							AssertEqual(t, allMap[idx][dbName], reflectValue.FieldByName(name).Interface())
						})
					}
				})
			}
		}
	})
}

func TestFindInBatches(t *testing.T) {
	var users = []User{
		*GetUser("find_in_batches", Config{}),
		*GetUser("find_in_batches", Config{}),
		*GetUser("find_in_batches", Config{}),
		*GetUser("find_in_batches", Config{}),
		*GetUser("find_in_batches", Config{}),
		*GetUser("find_in_batches", Config{}),
	}

	DB.Create(&users)

	var (
		results    []User
		totalBatch int
	)

	if result := DB.Where("name = ?", users[0].Name).FindInBatches(&results, 2, func(tx *gorm.DB, batch int) error {
		totalBatch += batch

		if tx.RowsAffected != 2 {
			t.Errorf("Incorrect affected rows, expects: 2, got %v", tx.RowsAffected)
		}

		if len(results) != 2 {
			t.Errorf("Incorrect users length, expects: 2, got %v", len(results))
		}

		return nil
	}); result.Error != nil || result.RowsAffected != 6 {
		t.Errorf("Failed to batch find, got error %v, rows affected: %v", result.Error, result.RowsAffected)
	}

	if totalBatch != 6 {
		t.Errorf("incorrect total batch, expects: %v, got %v", 6, totalBatch)
	}
}

func TestFillSmallerStruct(t *testing.T) {
	user := User{Name: "SmallerUser", Age: 100}
	DB.Save(&user)
	type SimpleUser struct {
		ID        int64
		CreatedAt time.Time
		UpdatedAt time.Time
		Name      string
	}

	var simpleUser SimpleUser
	if err := DB.Table("users").Where("name = ?", user.Name).First(&simpleUser).Error; err != nil {
		t.Fatalf("Failed to query smaller user, got error %v", err)
	}

	AssertObjEqual(t, user, simpleUser, "Name", "ID", "UpdatedAt", "CreatedAt")

	var simpleUser2 SimpleUser
	if err := DB.Model(&User{}).Select("id").First(&simpleUser2, user.ID).Error; err != nil {
		t.Fatalf("Failed to query smaller user, got error %v", err)
	}

	AssertObjEqual(t, user, simpleUser2, "ID")

	var simpleUsers []SimpleUser
	if err := DB.Model(&User{}).Select("id").Find(&simpleUsers, user.ID).Error; err != nil || len(simpleUsers) != 1 {
		t.Fatalf("Failed to query smaller user, got error %v", err)
	}

	AssertObjEqual(t, user, simpleUsers[0], "ID")

	result := DB.Session(&gorm.Session{DryRun: true}).Model(&User{}).Find(&simpleUsers, user.ID)

	if !regexp.MustCompile("SELECT .*id.*created_at.*updated_at.*name.* FROM .*users").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	result = DB.Session(&gorm.Session{DryRun: true}).Model(&User{}).Find(&User{}, user.ID)

	if regexp.MustCompile("SELECT .*name.* FROM .*users").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should not include selected names, but got %v", result.Statement.SQL.String())
	}

	result = DB.Session(&gorm.Session{DryRun: true}).Model(&User{}).Find(&[]User{}, user.ID)

	if regexp.MustCompile("SELECT .*name.* FROM .*users").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should not include selected names, but got %v", result.Statement.SQL.String())
	}

	result = DB.Session(&gorm.Session{DryRun: true}).Model(&User{}).Find(&[]*User{}, user.ID)

	if regexp.MustCompile("SELECT .*name.* FROM .*users").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should not include selected names, but got %v", result.Statement.SQL.String())
	}
}

func TestNot(t *testing.T) {
	dryDB := DB.Session(&gorm.Session{DryRun: true})

	result := dryDB.Not(map[string]interface{}{"name": "jinzhu"}).Find(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE .*name.* <> .+").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build NOT condition, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Where("name = ?", "jinzhu1").Not("name = ?", "jinzhu2").Find(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE .*name.* = .+ AND NOT.*name.* = .+").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build NOT condition, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Not("name = ?", "jinzhu").Find(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE NOT.*name.* = .+").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build NOT condition, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Not(map[string]interface{}{"name": []string{"jinzhu", "jinzhu 2"}}).Find(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE .*name.* NOT IN \\(.+,.+\\)").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build NOT condition, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Not([]int64{1, 2}).First(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE .*id.* NOT IN \\(.+,.+\\)").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build NOT condition, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Not([]int64{}).First(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE .users.\\..deleted_at. IS NULL ORDER BY").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build NOT condition, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Not(User{Name: "jinzhu", Age: 18}).First(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE .*users.*..*name.* <> .+ AND .*users.*..*age.* <> .+").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build NOT condition, but got %v", result.Statement.SQL.String())
	}
}

func TestOr(t *testing.T) {
	dryDB := DB.Session(&gorm.Session{DryRun: true})

	result := dryDB.Where("role = ?", "admin").Or("role = ?", "super_admin").Find(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE .*role.* = .+ OR .*role.* = .+").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build OR condition, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Where("name = ?", "jinzhu").Or(User{Name: "jinzhu 2", Age: 18}).Find(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE .*name.* = .+ OR \\(.*name.* AND .*age.*\\)").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build OR condition, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Where("name = ?", "jinzhu").Or(map[string]interface{}{"name": "jinzhu 2", "age": 18}).Find(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* WHERE .*name.* = .+ OR \\(.*age.* AND .*name.*\\)").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build OR condition, but got %v", result.Statement.SQL.String())
	}
}

func TestPluck(t *testing.T) {
	users := []*User{
		GetUser("pluck-user1", Config{}),
		GetUser("pluck-user2", Config{}),
		GetUser("pluck-user3", Config{}),
	}

	DB.Create(&users)

	var names []string
	if err := DB.Model(User{}).Where("name like ?", "pluck-user%").Order("name").Pluck("name", &names).Error; err != nil {
		t.Errorf("got error when pluck name: %v", err)
	}

	var names2 []string
	if err := DB.Model(User{}).Where("name like ?", "pluck-user%").Order("name desc").Pluck("name", &names2).Error; err != nil {
		t.Errorf("got error when pluck name: %v", err)
	}
	AssertEqual(t, names, sort.Reverse(sort.StringSlice(names2)))

	var ids []int
	if err := DB.Model(User{}).Where("name like ?", "pluck-user%").Pluck("id", &ids).Error; err != nil {
		t.Errorf("got error when pluck id: %v", err)
	}

	for idx, name := range names {
		if name != users[idx].Name {
			t.Errorf("Unexpected result on pluck name, got %+v", names)
		}
	}

	for idx, id := range ids {
		if int(id) != int(users[idx].ID) {
			t.Errorf("Unexpected result on pluck id, got %+v", ids)
		}
	}
}

func TestSelect(t *testing.T) {
	user := User{Name: "SelectUser1"}
	DB.Save(&user)

	var result User
	DB.Where("name = ?", user.Name).Select("name").Find(&result)
	if result.ID != 0 {
		t.Errorf("Should not have ID because only selected name, %+v", result.ID)
	}

	if user.Name != result.Name {
		t.Errorf("Should have user Name when selected it")
	}

	dryDB := DB.Session(&gorm.Session{DryRun: true})
	r := dryDB.Select("name", "age").Find(&User{})
	if !regexp.MustCompile("SELECT .*name.*,.*age.* FROM .*users.*").MatchString(r.Statement.SQL.String()) {
		t.Fatalf("Build Select with strings, but got %v", r.Statement.SQL.String())
	}

	r = dryDB.Select([]string{"name", "age"}).Find(&User{})
	if !regexp.MustCompile("SELECT .*name.*,.*age.* FROM .*users.*").MatchString(r.Statement.SQL.String()) {
		t.Fatalf("Build Select with slice, but got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("users").Select("COALESCE(age,?)", 42).Find(&User{})
	if !regexp.MustCompile(`SELECT COALESCE\(age,.*\) FROM .*users.*`).MatchString(r.Statement.SQL.String()) {
		t.Fatalf("Build Select with func, but got %v", r.Statement.SQL.String())
	}
	// SELECT COALESCE(age,'42') FROM users;

	r = dryDB.Select("u.*").Table("users as u").First(&User{}, user.ID)
	if !regexp.MustCompile(`SELECT u\.\* FROM .*users.*`).MatchString(r.Statement.SQL.String()) {
		t.Fatalf("Build Select with u.*, but got %v", r.Statement.SQL.String())
	}
}

func TestOmit(t *testing.T) {
	user := User{Name: "OmitUser1", Age: 20}
	DB.Save(&user)

	var result User
	DB.Where("name = ?", user.Name).Omit("name").Find(&result)
	if result.ID == 0 {
		t.Errorf("Should not have ID because only selected name, %+v", result.ID)
	}

	if result.Name != "" || result.Age != 20 {
		t.Errorf("User Name should be omitted, got %v, Age should be ok, got %v", result.Name, result.Age)
	}
}

func TestPluckWithSelect(t *testing.T) {
	users := []User{
		{Name: "pluck_with_select_1", Age: 25},
		{Name: "pluck_with_select_2", Age: 26},
	}

	DB.Create(&users)

	var userAges []int
	err := DB.Model(&User{}).Where("name like ?", "pluck_with_select%").Select("age + 1  as user_age").Pluck("user_age", &userAges).Error
	if err != nil {
		t.Fatalf("got error when pluck user_age: %v", err)
	}

	sort.Ints(userAges)

	AssertEqual(t, userAges, []int{26, 27})
}

func TestSelectWithVariables(t *testing.T) {
	DB.Save(&User{Name: "select_with_variables"})

	rows, _ := DB.Table("users").Where("name = ?", "select_with_variables").Select("? as fake", gorm.Expr("name")).Rows()

	if !rows.Next() {
		t.Errorf("Should have returned at least one row")
	} else {
		columns, _ := rows.Columns()
		AssertEqual(t, columns, []string{"fake"})
	}

	rows.Close()
}

func TestSelectWithArrayInput(t *testing.T) {
	DB.Save(&User{Name: "select_with_array", Age: 42})

	var user User
	DB.Select([]string{"name", "age"}).Where("age = 42 AND name = ?", "select_with_array").First(&user)

	if user.Name != "select_with_array" || user.Age != 42 {
		t.Errorf("Should have selected both age and name")
	}
}

func TestCustomizedTypePrimaryKey(t *testing.T) {
	type ID uint
	type CustomizedTypePrimaryKey struct {
		ID   ID
		Name string
	}

	DB.Migrator().DropTable(&CustomizedTypePrimaryKey{})
	if err := DB.AutoMigrate(&CustomizedTypePrimaryKey{}); err != nil {
		t.Fatalf("failed to migrate, got error %v", err)
	}

	p1 := CustomizedTypePrimaryKey{Name: "p1"}
	p2 := CustomizedTypePrimaryKey{Name: "p2"}
	p3 := CustomizedTypePrimaryKey{Name: "p3"}
	DB.Create(&p1)
	DB.Create(&p2)
	DB.Create(&p3)

	var p CustomizedTypePrimaryKey

	if err := DB.First(&p, p2.ID).Error; err != nil {
		t.Errorf("No error should returns, but got %v", err)
	}

	AssertEqual(t, p, p2)

	if err := DB.First(&p, "id = ?", p2.ID).Error; err != nil {
		t.Errorf("No error should happen when querying with customized type for primary key, got err %v", err)
	}

	AssertEqual(t, p, p2)
}

func TestStringPrimaryKeyForNumericValueStartingWithZero(t *testing.T) {
	type AddressByZipCode struct {
		ZipCode string `gorm:"primary_key"`
		Address string
	}

	DB.Migrator().DropTable(&AddressByZipCode{})
	if err := DB.AutoMigrate(&AddressByZipCode{}); err != nil {
		t.Fatalf("failed to migrate, got error %v", err)
	}

	address := AddressByZipCode{ZipCode: "00501", Address: "Holtsville"}
	DB.Create(&address)

	var result AddressByZipCode
	DB.First(&result, "00501")

	AssertEqual(t, result, address)
}

func TestSearchWithEmptyChain(t *testing.T) {
	user := User{Name: "search_with_empty_chain", Age: 1}
	DB.Create(&user)

	var result User
	if DB.Where("").Where("").First(&result).Error != nil {
		t.Errorf("Should not raise any error if searching with empty strings")
	}

	result = User{}
	if DB.Where(&User{}).Where("name = ?", user.Name).First(&result).Error != nil {
		t.Errorf("Should not raise any error if searching with empty struct")
	}

	result = User{}
	if DB.Where(map[string]interface{}{}).Where("name = ?", user.Name).First(&result).Error != nil {
		t.Errorf("Should not raise any error if searching with empty map")
	}
}

func TestOrder(t *testing.T) {
	dryDB := DB.Session(&gorm.Session{DryRun: true})

	result := dryDB.Order("age desc, name").Find(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* ORDER BY age desc, name").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build Order condition, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Order("age desc").Order("name").Find(&User{})
	if !regexp.MustCompile("SELECT \\* FROM .*users.* ORDER BY age desc,name").MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build Order condition, but got %v", result.Statement.SQL.String())
	}
}

func TestLimit(t *testing.T) {
	users := []User{
		{Name: "LimitUser1", Age: 1},
		{Name: "LimitUser2", Age: 10},
		{Name: "LimitUser3", Age: 20},
		{Name: "LimitUser4", Age: 10},
		{Name: "LimitUser5", Age: 20},
	}

	DB.Create(&users)

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
	DB.Where("name like ?", "OffsetUser%").Order("age desc").Find(&users1).Offset(3).Find(&users2).Offset(5).Find(&users3).Offset(-1).Find(&users4)

	if (len(users1) != len(users4)) || (len(users1)-len(users2) != 3) || (len(users1)-len(users3) != 5) {
		t.Errorf("Offset should work without limit.")
	}
}

func TestSearchWithMap(t *testing.T) {
	users := []User{
		*GetUser("map_search_user1", Config{}),
		*GetUser("map_search_user2", Config{}),
		*GetUser("map_search_user3", Config{}),
		*GetUser("map_search_user4", Config{Company: true}),
	}

	DB.Create(&users)

	var user User
	DB.First(&user, map[string]interface{}{"name": users[0].Name})
	CheckUser(t, user, users[0])

	user = User{}
	DB.Where(map[string]interface{}{"name": users[1].Name}).First(&user)
	CheckUser(t, user, users[1])

	var results []User
	DB.Where(map[string]interface{}{"name": users[2].Name}).Find(&results)
	if len(results) != 1 {
		t.Fatalf("Search all records with inline map")
	}

	CheckUser(t, results[0], users[2])

	var results2 []User
	DB.Find(&results2, map[string]interface{}{"name": users[3].Name, "company_id": nil})
	if len(results2) != 0 {
		t.Errorf("Search all records with inline map containing null value finding 0 records")
	}

	DB.Find(&results2, map[string]interface{}{"name": users[0].Name, "company_id": nil})
	if len(results2) != 1 {
		t.Errorf("Search all records with inline map containing null value finding 1 record")
	}

	DB.Find(&results2, map[string]interface{}{"name": users[3].Name, "company_id": users[3].CompanyID})
	if len(results2) != 1 {
		t.Errorf("Search all records with inline multiple value map")
	}
}

func TestSubQuery(t *testing.T) {
	users := []User{
		{Name: "subquery_1", Age: 10},
		{Name: "subquery_2", Age: 20},
		{Name: "subquery_3", Age: 30},
		{Name: "subquery_4", Age: 40},
	}

	DB.Create(&users)

	if err := DB.Select("*").Where("name IN (?)", DB.Select("name").Table("users").Where("name LIKE ?", "subquery_%")).Find(&users).Error; err != nil {
		t.Fatalf("got error: %v", err)
	}

	if len(users) != 4 {
		t.Errorf("Four users should be found, instead found %d", len(users))
	}

	DB.Select("*").Where("name LIKE ?", "subquery%").Where("age >= (?)", DB.
		Select("AVG(age)").Table("users").Where("name LIKE ?", "subquery%")).Find(&users)

	if len(users) != 2 {
		t.Errorf("Two users should be found, instead found %d", len(users))
	}
}

func TestSubQueryWithRaw(t *testing.T) {
	users := []User{
		{Name: "subquery_raw_1", Age: 10},
		{Name: "subquery_raw_2", Age: 20},
		{Name: "subquery_raw_3", Age: 30},
		{Name: "subquery_raw_4", Age: 40},
	}
	DB.Create(&users)

	var count int64
	err := DB.Raw("select count(*) from (?) tmp",
		DB.Table("users").
			Select("name").
			Where("age >= ? and name in (?)", 20, []string{"subquery_raw_1", "subquery_raw_3"}).
			Group("name"),
	).Count(&count).Error

	if err != nil {
		t.Errorf("Expected to get no errors, but got %v", err)
	}

	if count != 1 {
		t.Errorf("Row count must be 1, instead got %d", count)
	}

	err = DB.Raw("select count(*) from (?) tmp",
		DB.Table("users").
			Select("name").
			Where("name LIKE ?", "subquery_raw%").
			Not("age <= ?", 10).Not("name IN (?)", []string{"subquery_raw_1", "subquery_raw_3"}).
			Group("name"),
	).Count(&count).Error

	if err != nil {
		t.Errorf("Expected to get no errors, but got %v", err)
	}

	if count != 2 {
		t.Errorf("Row count must be 2, instead got %d", count)
	}
}

func TestSubQueryWithHaving(t *testing.T) {
	users := []User{
		{Name: "subquery_having_1", Age: 10},
		{Name: "subquery_having_2", Age: 20},
		{Name: "subquery_having_3", Age: 30},
		{Name: "subquery_having_4", Age: 40},
	}
	DB.Create(&users)

	var results []User
	DB.Select("AVG(age) as avgage").Where("name LIKE ?", "subquery_having%").Group("name").Having("AVG(age) > (?)", DB.
		Select("AVG(age)").Where("name LIKE ?", "subquery_having%").Table("users")).Find(&results)

	if len(results) != 2 {
		t.Errorf("Two user group should be found, instead found %d", len(results))
	}
}

func TestScanNullValue(t *testing.T) {
	user := GetUser("scan_null_value", Config{})
	DB.Create(&user)

	if err := DB.Model(&user).Update("age", nil).Error; err != nil {
		t.Fatalf("failed to update column age for struct, got error %v", err)
	}

	var result User
	if err := DB.First(&result, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("failed to query struct data with null age, got error %v", err)
	}

	AssertEqual(t, result, user)

	users := []User{
		*GetUser("scan_null_value_for_slice_1", Config{}),
		*GetUser("scan_null_value_for_slice_2", Config{}),
		*GetUser("scan_null_value_for_slice_3", Config{}),
	}
	DB.Create(&users)

	if err := DB.Model(&users[0]).Update("age", nil).Error; err != nil {
		t.Fatalf("failed to update column age for struct, got error %v", err)
	}

	var results []User
	if err := DB.Find(&results, "name like ?", "scan_null_value_for_slice%").Error; err != nil {
		t.Fatalf("failed to query slice data with null age, got error %v", err)
	}
}
