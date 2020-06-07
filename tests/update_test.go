package tests_test

import (
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestUpdate(t *testing.T) {
	var (
		users = []*User{
			GetUser("update-1", Config{}),
			GetUser("update-2", Config{}),
			GetUser("update-3", Config{}),
		}
		user          = users[1]
		lastUpdatedAt time.Time
	)

	checkUpdatedAtChanged := func(name string, n time.Time) {
		if n.UnixNano() == lastUpdatedAt.UnixNano() {
			t.Errorf("%v: user's updated at should be changed, but got %v, was %v", name, n, lastUpdatedAt)
		}
		lastUpdatedAt = n
	}

	checkOtherData := func(name string) {
		var first, last User
		if err := DB.Where("id = ?", users[0].ID).First(&first).Error; err != nil {
			t.Errorf("errors happened when query before user: %v", err)
		}
		CheckUser(t, first, *users[0])

		if err := DB.Where("id = ?", users[2].ID).First(&last).Error; err != nil {
			t.Errorf("errors happened when query after user: %v", err)
		}
		CheckUser(t, last, *users[2])
	}

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	} else if user.ID == 0 {
		t.Fatalf("user's primary value should not zero, %v", user.ID)
	} else if user.UpdatedAt.IsZero() {
		t.Fatalf("user's updated at should not zero, %v", user.UpdatedAt)
	}
	lastUpdatedAt = user.UpdatedAt

	if err := DB.Model(user).Update("Age", 10).Error; err != nil {
		t.Errorf("errors happened when update: %v", err)
	} else if user.Age != 10 {
		t.Errorf("Age should equals to 10, but got %v", user.Age)
	}
	checkUpdatedAtChanged("Update", user.UpdatedAt)
	checkOtherData("Update")

	var result User
	if err := DB.Where("id = ?", user.ID).First(&result).Error; err != nil {
		t.Errorf("errors happened when query: %v", err)
	} else {
		CheckUser(t, result, *user)
	}

	values := map[string]interface{}{"Active": true, "age": 5}
	if err := DB.Model(user).Updates(values).Error; err != nil {
		t.Errorf("errors happened when update: %v", err)
	} else if user.Age != 5 {
		t.Errorf("Age should equals to 5, but got %v", user.Age)
	} else if user.Active != true {
		t.Errorf("Active should be true, but got %v", user.Active)
	}
	checkUpdatedAtChanged("Updates with map", user.UpdatedAt)
	checkOtherData("Updates with map")

	var result2 User
	if err := DB.Where("id = ?", user.ID).First(&result2).Error; err != nil {
		t.Errorf("errors happened when query: %v", err)
	} else {
		CheckUser(t, result2, *user)
	}

	if err := DB.Model(user).Updates(User{Age: 2}).Error; err != nil {
		t.Errorf("errors happened when update: %v", err)
	} else if user.Age != 2 {
		t.Errorf("Age should equals to 2, but got %v", user.Age)
	}
	checkUpdatedAtChanged("Updates with struct", user.UpdatedAt)
	checkOtherData("Updates with struct")

	var result3 User
	if err := DB.Where("id = ?", user.ID).First(&result3).Error; err != nil {
		t.Errorf("errors happened when query: %v", err)
	} else {
		CheckUser(t, result3, *user)
	}

	user.Active = false
	user.Age = 1
	if err := DB.Save(user).Error; err != nil {
		t.Errorf("errors happened when update: %v", err)
	} else if user.Age != 1 {
		t.Errorf("Age should equals to 1, but got %v", user.Age)
	} else if user.Active != false {
		t.Errorf("Active should equals to false, but got %v", user.Active)
	}
	checkUpdatedAtChanged("Save", user.UpdatedAt)
	checkOtherData("Save")

	var result4 User
	if err := DB.Where("id = ?", user.ID).First(&result4).Error; err != nil {
		t.Errorf("errors happened when query: %v", err)
	} else {
		CheckUser(t, result4, *user)
	}
}

func TestUpdates(t *testing.T) {
	var users = []*User{
		GetUser("updates_01", Config{}),
		GetUser("updates_02", Config{}),
	}

	DB.Create(&users)
	lastUpdatedAt := users[0].UpdatedAt

	// update with map
	DB.Model(users[0]).Updates(map[string]interface{}{"name": "updates_01_newname", "age": 100})
	if users[0].Name != "updates_01_newname" || users[0].Age != 100 {
		t.Errorf("Record should be updated also with map")
	}

	if users[0].UpdatedAt.UnixNano() == lastUpdatedAt.UnixNano() {
		t.Errorf("User's updated at should be changed, but got %v, was %v", users[0].UpdatedAt.UnixNano(), lastUpdatedAt)
	}

	// user2 should not be updated
	var user1, user2 User
	DB.First(&user1, users[0].ID)
	DB.First(&user2, users[1].ID)
	CheckUser(t, user1, *users[0])
	CheckUser(t, user2, *users[1])

	// update with struct
	DB.Table("users").Where("name in ?", []string{users[1].Name}).Updates(User{Name: "updates_02_newname"})

	var user3 User
	if err := DB.First(&user3, "name = ?", "updates_02_newname").Error; err != nil {
		t.Errorf("User2's name should be updated")
	}
	AssertEqual(t, user2.UpdatedAt, user3.UpdatedAt)

	// update with gorm exprs
	if err := DB.Model(&user3).Updates(map[string]interface{}{"age": gorm.Expr("age + ?", 100)}).Error; err != nil {
		t.Errorf("Not error should happen when updating with gorm expr, but got %v", err)
	}
	var user4 User
	DB.First(&user4, user3.ID)

	user3.Age += 100
	AssertObjEqual(t, user4, user3, "UpdatedAt", "Age")
}

func TestUpdateColumn(t *testing.T) {
	var users = []*User{
		GetUser("update_column_01", Config{}),
		GetUser("update_column_02", Config{}),
	}

	DB.Create(&users)
	lastUpdatedAt := users[1].UpdatedAt

	// update with map
	DB.Model(users[1]).UpdateColumns(map[string]interface{}{"name": "update_column_02_newname", "age": 100})
	if users[1].Name != "update_column_02_newname" || users[1].Age != 100 {
		t.Errorf("user 2 should be updated with update column")
	}
	AssertEqual(t, lastUpdatedAt.UnixNano(), users[1].UpdatedAt.UnixNano())

	// user2 should not be updated
	var user1, user2 User
	DB.First(&user1, users[0].ID)
	DB.First(&user2, users[1].ID)
	CheckUser(t, user1, *users[0])
	CheckUser(t, user2, *users[1])

	DB.Model(users[1]).UpdateColumn("name", "update_column_02_newnew")
	AssertEqual(t, lastUpdatedAt.UnixNano(), users[1].UpdatedAt.UnixNano())

	if users[1].Name != "update_column_02_newnew" {
		t.Errorf("user 2's name should be updated, but got %v", users[1].Name)
	}

	DB.Model(users[1]).UpdateColumn("age", gorm.Expr("age + 100 - 50"))
	var user3 User
	DB.First(&user3, users[1].ID)

	users[1].Age += 50
	CheckUser(t, user3, *users[1])

	// update with struct
	DB.Model(users[1]).UpdateColumns(User{Name: "update_column_02_newnew2", Age: 200})
	if users[1].Name != "update_column_02_newnew2" || users[1].Age != 200 {
		t.Errorf("user 2 should be updated with update column")
	}
	AssertEqual(t, lastUpdatedAt.UnixNano(), users[1].UpdatedAt.UnixNano())

	// user2 should not be updated
	var user5, user6 User
	DB.First(&user5, users[0].ID)
	DB.First(&user6, users[1].ID)
	CheckUser(t, user5, *users[0])
	CheckUser(t, user6, *users[1])
}

func TestBlockGlobalUpdate(t *testing.T) {
	if err := DB.Model(&User{}).Update("name", "jinzhu").Error; err == nil || !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Errorf("should returns missing WHERE clause while updating error, got err %v", err)
	}
}

func TestSelectWithUpdate(t *testing.T) {
	user := *GetUser("select_update", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	DB.Create(&user)

	var result User
	DB.First(&result, user.ID)

	user2 := *GetUser("select_update_new", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	result.Name = user2.Name
	result.Age = 50
	result.Account = user2.Account
	result.Pets = user2.Pets
	result.Toys = user2.Toys
	result.Company = user2.Company
	result.Manager = user2.Manager
	result.Team = user2.Team
	result.Languages = user2.Languages
	result.Friends = user2.Friends

	DB.Select("Name", "Account", "Toys", "Manager", "ManagerID", "Languages").Save(&result)

	var result2 User
	DB.Preload("Account").Preload("Pets").Preload("Toys").Preload("Company").Preload("Manager").Preload("Team").Preload("Languages").Preload("Friends").First(&result2, user.ID)

	result.Languages = append(user.Languages, result.Languages...)
	result.Toys = append(user.Toys, result.Toys...)

	sort.Slice(result.Languages, func(i, j int) bool {
		return strings.Compare(result.Languages[i].Code, result.Languages[j].Code) > 0
	})

	sort.Slice(result.Toys, func(i, j int) bool {
		return result.Toys[i].ID < result.Toys[j].ID
	})

	sort.Slice(result2.Languages, func(i, j int) bool {
		return strings.Compare(result2.Languages[i].Code, result2.Languages[j].Code) > 0
	})

	sort.Slice(result2.Toys, func(i, j int) bool {
		return result2.Toys[i].ID < result2.Toys[j].ID
	})

	AssertObjEqual(t, result2, result, "Name", "Account", "Toys", "Manager", "ManagerID", "Languages")
}

func TestSelectWithUpdateWithMap(t *testing.T) {
	user := *GetUser("select_update_map", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	DB.Create(&user)

	var result User
	DB.First(&result, user.ID)

	user2 := *GetUser("select_update_map_new", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	updateValues := map[string]interface{}{
		"Name":      user2.Name,
		"Age":       50,
		"Account":   user2.Account,
		"Pets":      user2.Pets,
		"Toys":      user2.Toys,
		"Company":   user2.Company,
		"Manager":   user2.Manager,
		"Team":      user2.Team,
		"Languages": user2.Languages,
		"Friends":   user2.Friends,
	}

	DB.Model(&result).Select("Name", "Account", "Toys", "Manager", "ManagerID", "Languages").Updates(updateValues)

	var result2 User
	DB.Preload("Account").Preload("Pets").Preload("Toys").Preload("Company").Preload("Manager").Preload("Team").Preload("Languages").Preload("Friends").First(&result2, user.ID)

	result.Languages = append(user.Languages, result.Languages...)
	result.Toys = append(user.Toys, result.Toys...)

	sort.Slice(result.Languages, func(i, j int) bool {
		return strings.Compare(result.Languages[i].Code, result.Languages[j].Code) > 0
	})

	sort.Slice(result.Toys, func(i, j int) bool {
		return result.Toys[i].ID < result.Toys[j].ID
	})

	sort.Slice(result2.Languages, func(i, j int) bool {
		return strings.Compare(result2.Languages[i].Code, result2.Languages[j].Code) > 0
	})

	sort.Slice(result2.Toys, func(i, j int) bool {
		return result2.Toys[i].ID < result2.Toys[j].ID
	})

	AssertObjEqual(t, result2, result, "Name", "Account", "Toys", "Manager", "ManagerID", "Languages")
}

func TestOmitWithUpdate(t *testing.T) {
	user := *GetUser("omit_update", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	DB.Create(&user)

	var result User
	DB.First(&result, user.ID)

	user2 := *GetUser("omit_update_new", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	result.Name = user2.Name
	result.Age = 50
	result.Account = user2.Account
	result.Pets = user2.Pets
	result.Toys = user2.Toys
	result.Company = user2.Company
	result.Manager = user2.Manager
	result.Team = user2.Team
	result.Languages = user2.Languages
	result.Friends = user2.Friends

	DB.Omit("Name", "Account", "Toys", "Manager", "ManagerID", "Languages").Save(&result)

	var result2 User
	DB.Preload("Account").Preload("Pets").Preload("Toys").Preload("Company").Preload("Manager").Preload("Team").Preload("Languages").Preload("Friends").First(&result2, user.ID)

	result.Pets = append(user.Pets, result.Pets...)
	result.Team = append(user.Team, result.Team...)
	result.Friends = append(user.Friends, result.Friends...)

	sort.Slice(result.Pets, func(i, j int) bool {
		return result.Pets[i].ID < result.Pets[j].ID
	})
	sort.Slice(result.Team, func(i, j int) bool {
		return result.Team[i].ID < result.Team[j].ID
	})
	sort.Slice(result.Friends, func(i, j int) bool {
		return result.Friends[i].ID < result.Friends[j].ID
	})
	sort.Slice(result2.Pets, func(i, j int) bool {
		return result2.Pets[i].ID < result2.Pets[j].ID
	})
	sort.Slice(result2.Team, func(i, j int) bool {
		return result2.Team[i].ID < result2.Team[j].ID
	})
	sort.Slice(result2.Friends, func(i, j int) bool {
		return result2.Friends[i].ID < result2.Friends[j].ID
	})

	AssertObjEqual(t, result2, result, "Age", "Pets", "Company", "CompanyID", "Team", "Friends")
}

func TestOmitWithUpdateWithMap(t *testing.T) {
	user := *GetUser("omit_update_map", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	DB.Create(&user)

	var result User
	DB.First(&result, user.ID)

	user2 := *GetUser("omit_update_map_new", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	updateValues := map[string]interface{}{
		"Name":      user2.Name,
		"Age":       50,
		"Account":   user2.Account,
		"Pets":      user2.Pets,
		"Toys":      user2.Toys,
		"Company":   user2.Company,
		"Manager":   user2.Manager,
		"Team":      user2.Team,
		"Languages": user2.Languages,
		"Friends":   user2.Friends,
	}

	DB.Model(&result).Omit("Name", "Account", "Toys", "Manager", "ManagerID", "Languages").Updates(updateValues)

	var result2 User
	DB.Preload("Account").Preload("Pets").Preload("Toys").Preload("Company").Preload("Manager").Preload("Team").Preload("Languages").Preload("Friends").First(&result2, user.ID)

	result.Pets = append(user.Pets, result.Pets...)
	result.Team = append(user.Team, result.Team...)
	result.Friends = append(user.Friends, result.Friends...)

	sort.Slice(result.Pets, func(i, j int) bool {
		return result.Pets[i].ID < result.Pets[j].ID
	})
	sort.Slice(result.Team, func(i, j int) bool {
		return result.Team[i].ID < result.Team[j].ID
	})
	sort.Slice(result.Friends, func(i, j int) bool {
		return result.Friends[i].ID < result.Friends[j].ID
	})
	sort.Slice(result2.Pets, func(i, j int) bool {
		return result2.Pets[i].ID < result2.Pets[j].ID
	})
	sort.Slice(result2.Team, func(i, j int) bool {
		return result2.Team[i].ID < result2.Team[j].ID
	})
	sort.Slice(result2.Friends, func(i, j int) bool {
		return result2.Friends[i].ID < result2.Friends[j].ID
	})

	AssertObjEqual(t, result2, result, "Age", "Pets", "Company", "CompanyID", "Team", "Friends")
}

func TestSelectWithUpdateColumn(t *testing.T) {
	user := *GetUser("select_with_update_column", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	DB.Create(&user)

	updateValues := map[string]interface{}{"Name": "new_name", "Age": 50}

	var result User
	DB.First(&result, user.ID)
	DB.Model(&result).Select("Name").UpdateColumns(updateValues)

	var result2 User
	DB.First(&result2, user.ID)

	if result2.Name == user.Name || result2.Age != user.Age {
		t.Errorf("Should only update users with name column")
	}
}

func TestOmitWithUpdateColumn(t *testing.T) {
	user := *GetUser("omit_with_update_column", Config{Account: true, Pets: 3, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 4})
	DB.Create(&user)

	updateValues := map[string]interface{}{"Name": "new_name", "Age": 50}

	var result User
	DB.First(&result, user.ID)
	DB.Model(&result).Omit("Name").UpdateColumns(updateValues)

	var result2 User
	DB.First(&result2, user.ID)

	if result2.Name != user.Name || result2.Age == user.Age {
		t.Errorf("Should only update users with name column")
	}
}

func TestUpdateColumnsSkipsAssociations(t *testing.T) {
	user := *GetUser("update_column_skips_association", Config{})
	DB.Create(&user)

	// Update a single field of the user and verify that the changed address is not stored.
	newAge := uint(100)
	user.Account.Number = "new_account_number"
	db := DB.Model(&user).UpdateColumns(User{Age: newAge})

	if db.RowsAffected != 1 {
		t.Errorf("Expected RowsAffected=1 but instead RowsAffected=%v", db.RowsAffected)
	}

	// Verify that Age now=`newAge`.
	result := &User{}
	result.ID = user.ID
	DB.Preload("Account").First(result)

	if result.Age != newAge {
		t.Errorf("Expected freshly queried user to have Age=%v but instead found Age=%v", newAge, result.Age)
	}

	if result.Account.Number != user.Account.Number {
		t.Errorf("account number should not been changed, expects: %v, got %v", user.Account.Number, result.Account.Number)
	}
}

func TestUpdatesWithBlankValues(t *testing.T) {
	user := *GetUser("updates_with_blank_value", Config{})
	DB.Save(&user)

	var user2 User
	user2.ID = user.ID
	DB.Model(&user2).Updates(&User{Age: 100})

	var result User
	DB.First(&result, user.ID)

	if result.Name != user.Name || result.Age != 100 {
		t.Errorf("user's name should not be updated")
	}
}

func TestUpdatesTableWithIgnoredValues(t *testing.T) {
	type ElementWithIgnoredField struct {
		Id           int64
		Value        string
		IgnoredField int64 `gorm:"-"`
	}
	DB.Migrator().DropTable(&ElementWithIgnoredField{})
	DB.AutoMigrate(&ElementWithIgnoredField{})

	elem := ElementWithIgnoredField{Value: "foo", IgnoredField: 10}
	DB.Save(&elem)

	DB.Model(&ElementWithIgnoredField{}).
		Where("id = ?", elem.Id).
		Updates(&ElementWithIgnoredField{Value: "bar", IgnoredField: 100})

	var result ElementWithIgnoredField
	if err := DB.First(&result, elem.Id).Error; err != nil {
		t.Errorf("error getting an element from database: %s", err.Error())
	}

	if result.IgnoredField != 0 {
		t.Errorf("element's ignored field should not be updated")
	}
}
