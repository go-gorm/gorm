package tests_test

import (
	"regexp"
	"sort"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestJoins(t *testing.T) {
	user := *GetUser("joins-1", Config{Company: true, Manager: true, Account: true})

	DB.Create(&user)

	var user2 User
	if err := DB.Joins("Company").Joins("Manager").Joins("Account").First(&user2, "users.name = ?", user.Name).Error; err != nil {
		t.Fatalf("Failed to load with joins, got error: %v", err)
	}

	CheckUser(t, user2, user)
}

func TestJoinsForSlice(t *testing.T) {
	users := []User{
		*GetUser("slice-joins-1", Config{Company: true, Manager: true, Account: true}),
		*GetUser("slice-joins-2", Config{Company: true, Manager: true, Account: true}),
		*GetUser("slice-joins-3", Config{Company: true, Manager: true, Account: true}),
	}

	DB.Create(&users)

	var userIDs []uint
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	var users2 []User
	if err := DB.Joins("Company").Joins("Manager").Joins("Account").Find(&users2, "users.id IN ?", userIDs).Error; err != nil {
		t.Fatalf("Failed to load with joins, got error: %v", err)
	} else if len(users2) != len(users) {
		t.Fatalf("Failed to load join users, got: %v, expect: %v", len(users2), len(users))
	}

	sort.Slice(users2, func(i, j int) bool {
		return users2[i].ID > users2[j].ID
	})

	sort.Slice(users, func(i, j int) bool {
		return users[i].ID > users[j].ID
	})

	for idx, user := range users {
		CheckUser(t, user, users2[idx])
	}
}

func TestJoinConds(t *testing.T) {
	var user = *GetUser("joins-conds", Config{Account: true, Pets: 3})
	DB.Save(&user)

	var users1 []User
	DB.Joins("inner join pets on pets.user_id = users.id").Where("users.name = ?", user.Name).Find(&users1)
	if len(users1) != 3 {
		t.Errorf("should find two users using left join, but got %v", len(users1))
	}

	var users2 []User
	DB.Joins("inner join pets on pets.user_id = users.id AND pets.name = ?", user.Pets[0].Name).Where("users.name = ?", user.Name).First(&users2)
	if len(users2) != 1 {
		t.Errorf("should find one users using left join with conditions, but got %v", len(users2))
	}

	var users3 []User
	DB.Joins("inner join pets on pets.user_id = users.id AND pets.name = ?", user.Pets[0].Name).Joins("join accounts on accounts.user_id = users.id AND accounts.number = ?", user.Account.Number).Where("users.name = ?", user.Name).First(&users3)
	if len(users3) != 1 {
		t.Errorf("should find one users using multiple left join conditions, but got %v", len(users3))
	}

	var users4 []User
	DB.Joins("inner join pets on pets.user_id = users.id AND pets.name = ?", user.Pets[0].Name).Joins("join accounts on accounts.user_id = users.id AND accounts.number = ?", user.Account.Number+"non-exist").Where("users.name = ?", user.Name).First(&users4)
	if len(users4) != 0 {
		t.Errorf("should find no user when searching with unexisting credit card, but got %v", len(users4))
	}

	var users5 []User
	db5 := DB.Joins("inner join pets on pets.user_id = users.id AND pets.name = ?", user.Pets[0].Name).Joins("join accounts on accounts.user_id = users.id AND accounts.number = ?", user.Account.Number).Where(User{Model: gorm.Model{ID: 1}}).Where(Account{Model: gorm.Model{ID: 1}}).Not(Pet{Model: gorm.Model{ID: 1}}).Find(&users5)
	if db5.Error != nil {
		t.Errorf("Should not raise error for join where identical fields in different tables. Error: %s", db5.Error.Error())
	}

	var users6 []User
	DB.Joins("inner join pets on pets.user_id = users.id AND pets.name = @Name", user.Pets[0]).Where("users.name = ?", user.Name).First(&users6)
	if len(users6) != 1 {
		t.Errorf("should find one users using left join with conditions, but got %v", len(users6))
	}

	dryDB := DB.Session(&gorm.Session{DryRun: true})
	stmt := dryDB.Joins("left join pets on pets.user_id = users.id AND pets.name = ?", user.Pets[0].Name).Joins("join accounts on accounts.user_id = users.id AND accounts.number = ?", user.Account.Number).Where(User{Model: gorm.Model{ID: 1}}).Where(Account{Model: gorm.Model{ID: 1}}).Not(Pet{Model: gorm.Model{ID: 1}}).Find(&users5).Statement

	if !regexp.MustCompile("SELECT .* FROM .users. left join pets.*join accounts.*").MatchString(stmt.SQL.String()) {
		t.Errorf("joins should be ordered, but got %v", stmt.SQL.String())
	}

	iv := DB.Table(`table_invoices`).Select(`seller, SUM(total) as total, SUM(paid) as paid, SUM(balance) as balance`).Group(`seller`)
	stmt = dryDB.Table(`table_employees`).Select(`id, name, iv.total, iv.paid, iv.balance`).Joins(`LEFT JOIN (?) AS iv ON iv.seller = table_employees.id`, iv).Scan(&user).Statement
	if !regexp.MustCompile("SELECT id, name, iv.total, iv.paid, iv.balance FROM .table_employees. LEFT JOIN \\(SELECT seller, SUM\\(total\\) as total, SUM\\(paid\\) as paid, SUM\\(balance\\) as balance FROM .table_invoices. GROUP BY .seller.\\) AS iv ON iv.seller = table_employees.id").MatchString(stmt.SQL.String()) {
		t.Errorf("joins should be ordered, but got %v", stmt.SQL.String())
	}
}

func TestJoinOn(t *testing.T) {
	var user = *GetUser("joins-on", Config{Pets: 2})
	DB.Save(&user)

	var user1 User
	onQuery := DB.Where(&Pet{Name: "joins-on_pet_1"})

	if err := DB.Joins("NamedPet", onQuery).Where("users.name = ?", user.Name).First(&user1).Error; err != nil {
		t.Fatalf("Failed to load with joins on, got error: %v", err)
	}

	AssertEqual(t, user1.NamedPet.Name, "joins-on_pet_1")

	onQuery2 := DB.Where(&Pet{Name: "joins-on_pet_2"})
	var user2 User
	if err := DB.Joins("NamedPet", onQuery2).Where("users.name = ?", user.Name).First(&user2).Error; err != nil {
		t.Fatalf("Failed to load with joins on, got error: %v", err)
	}
	AssertEqual(t, user2.NamedPet.Name, "joins-on_pet_2")
}

func TestJoinsWithSelect(t *testing.T) {
	type result struct {
		ID    uint
		PetID uint
		Name  string
	}

	user := *GetUser("joins_with_select", Config{Pets: 2})
	DB.Save(&user)

	var results []result

	DB.Table("users").Select("users.id, pets.id as pet_id, pets.name").Joins("left join pets on pets.user_id = users.id").Where("users.name = ?", "joins_with_select").Scan(&results)

	sort.Slice(results, func(i, j int) bool {
		return results[i].PetID > results[j].PetID
	})

	sort.Slice(results, func(i, j int) bool {
		return user.Pets[i].ID > user.Pets[j].ID
	})

	if len(results) != 2 || results[0].Name != user.Pets[0].Name || results[1].Name != user.Pets[1].Name {
		t.Errorf("Should find all two pets with Join select, got %+v", results)
	}
}

func TestJoinCount(t *testing.T) {
	companyA := Company{Name: "A"}
	companyB := Company{Name: "B"}
	DB.Create(&companyA)
	DB.Create(&companyB)

	user := User{Name: "kingGo", CompanyID: &companyB.ID}
	DB.Create(&user)

	query := DB.Model(&User{}).Joins("Company")
	//Bug happens when .Count is called on a query.
	//Removing the below two lines or downgrading to gorm v1.20.12 will make this test pass.
	var total int64
	query.Count(&total)

	var result User

	// Incorrectly generates a 'SELECT *' query which causes companies.id to overwrite users.id
	if err := query.First(&result, user.ID).Error; err != nil {
		t.Fatalf("Failed, got error: %v", err)
	}

	if result.ID != user.ID {
		t.Fatalf("result's id, %d, doesn't match user's id, %d", result.ID, user.ID)
	}
}
