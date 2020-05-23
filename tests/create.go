package tests

import (
	"strconv"
	"testing"
	"time"
)

type Config struct {
	Account   bool
	Pets      int
	Toys      int
	Company   bool
	Manager   bool
	Team      int
	Languages int
	Friends   int
}

func GetUser(name string, config Config) *User {
	var (
		birthday = time.Now()
		user     = User{
			Name:     name,
			Age:      18,
			Birthday: &birthday,
		}
	)

	if config.Account {
		user.Account = Account{Number: name + "_account"}
	}

	for i := 0; i < config.Pets; i++ {
		user.Pets = append(user.Pets, &Pet{Name: name + "_pet_" + strconv.Itoa(i+1)})
	}

	for i := 0; i < config.Toys; i++ {
		user.Toys = append(user.Toys, Toy{Name: name + "_toy_" + strconv.Itoa(i+1)})
	}

	if config.Company {
		user.Company = Company{Name: "company-" + name}
	}

	if config.Manager {
		user.Manager = GetUser(name+"_manager", Config{})
	}

	for i := 0; i < config.Team; i++ {
		user.Team = append(user.Team, *GetUser(name+"_team_"+strconv.Itoa(i+1), Config{}))
	}

	for i := 0; i < config.Languages; i++ {
		name := name + "_locale_" + strconv.Itoa(i+1)
		language := Language{Code: name, Name: name}
		DB.Create(&language)
		user.Languages = append(user.Languages, language)
	}

	for i := 0; i < config.Friends; i++ {
		user.Friends = append(user.Friends, GetUser(name+"_friend_"+strconv.Itoa(i+1), Config{}))
	}

	return &user
}

func CheckPet(t *testing.T, pet Pet, expect Pet) {
	if pet.ID != 0 {
		var newPet Pet
		if err := DB.Where("id = ?", pet.ID).First(&newPet).Error; err != nil {
			t.Fatalf("errors happened when query: %v", err)
		} else {
			AssertObjEqual(t, newPet, pet, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "UserID", "Name")
		}
	}

	AssertObjEqual(t, pet, expect, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "UserID", "Name")

	AssertObjEqual(t, pet.Toy, expect.Toy, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Name", "OwnerID", "OwnerType")

	if expect.Toy.Name != "" && expect.Toy.OwnerType != "pets" {
		t.Errorf("toys's OwnerType, expect: %v, got %v", "pets", expect.Toy.OwnerType)
	}
}

func CheckUser(t *testing.T, user User, expect User) {
	if user.ID != 0 {
		var newUser User
		if err := DB.Where("id = ?", user.ID).First(&newUser).Error; err != nil {
			t.Fatalf("errors happened when query: %v", err)
		} else {
			AssertObjEqual(t, newUser, user, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Name", "Age", "Birthday", "CompanyID", "ManagerID", "Active")
		}
	}

	AssertObjEqual(t, user, expect, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Name", "Age", "Birthday", "CompanyID", "ManagerID", "Active")

	t.Run("Account", func(t *testing.T) {
		AssertObjEqual(t, user.Account, expect.Account, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "UserID", "Number")

		if user.Account.Number != "" {
			if !user.Account.UserID.Valid {
				t.Errorf("Account's foreign key should be saved")
			} else {
				var account Account
				DB.First(&account, "user_id = ?", user.ID)
				AssertObjEqual(t, account, user.Account, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "UserID", "Number")
			}
		}
	})

	t.Run("Pets", func(t *testing.T) {
		if len(user.Pets) != len(expect.Pets) {
			t.Errorf("pets should equal, expect: %v, got %v", len(expect.Pets), len(user.Pets))
		}

		for idx, pet := range user.Pets {
			if pet == nil || expect.Pets[idx] == nil {
				t.Errorf("pets#%v should equal, expect: %v, got %v", idx, expect.Pets[idx], pet)
			} else {
				CheckPet(t, *pet, *expect.Pets[idx])
			}
		}
	})

	t.Run("Toys", func(t *testing.T) {
		if len(user.Toys) != len(expect.Toys) {
			t.Errorf("toys should equal, expect: %v, got %v", len(expect.Toys), len(user.Toys))
		}

		for idx, toy := range user.Toys {
			if toy.OwnerType != "users" {
				t.Errorf("toys's OwnerType, expect: %v, got %v", "users", toy.OwnerType)
			}

			AssertObjEqual(t, toy, expect.Toys[idx], "ID", "CreatedAt", "UpdatedAt", "Name", "OwnerID", "OwnerType")
		}
	})

	t.Run("Company", func(t *testing.T) {
		AssertObjEqual(t, user.Company, expect.Company, "ID", "Name")
	})

	t.Run("Manager", func(t *testing.T) {
		if user.Manager != nil {
			if user.ManagerID == nil {
				t.Errorf("Manager's foreign key should be saved")
			} else {
				var manager User
				DB.First(&manager, "id = ?", *user.ManagerID)
				AssertObjEqual(t, manager, user.Manager, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Name", "Age", "Birthday", "CompanyID", "ManagerID", "Active")
			}
		} else if user.ManagerID != nil {
			t.Errorf("Manager should not be created for zero value, got: %+v", user.ManagerID)
		}
	})

	t.Run("Team", func(t *testing.T) {
		if len(user.Team) != len(expect.Team) {
			t.Errorf("Team should equal, expect: %v, got %v", len(expect.Team), len(user.Team))
		}

		for idx, team := range user.Team {
			AssertObjEqual(t, team, expect.Team[idx], "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Name", "Age", "Birthday", "CompanyID", "ManagerID", "Active")
		}
	})

	t.Run("Languages", func(t *testing.T) {
		if len(user.Languages) != len(expect.Languages) {
			t.Errorf("Languages should equal, expect: %v, got %v", len(expect.Languages), len(user.Languages))
		}

		for idx, language := range user.Languages {
			AssertObjEqual(t, language, expect.Languages[idx], "Code", "Name")
		}
	})

	t.Run("Friends", func(t *testing.T) {
		if len(user.Friends) != len(expect.Friends) {
			t.Errorf("Friends should equal, expect: %v, got %v", len(expect.Friends), len(user.Friends))
		}

		for idx, friend := range user.Friends {
			AssertObjEqual(t, friend, expect.Friends[idx], "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Name", "Age", "Birthday", "CompanyID", "ManagerID", "Active")
		}
	})
}
