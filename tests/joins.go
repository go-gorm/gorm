package tests

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestJoins(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&User{}, &Account{}, &Company{})
	db.AutoMigrate(&User{}, &Account{}, &Company{})

	check := func(t *testing.T, oldUser, newUser User) {
		if newUser.Company.ID != oldUser.Company.ID {
			t.Errorf("Company is not equal when load with joins, loaded company id: %v", newUser.Company.ID)
		}

		if newUser.Manager == nil || newUser.Manager.ID != oldUser.Manager.ID {
			t.Errorf("Manager is not equal when load with joins: loaded manager: %+v", newUser.Manager)
		}

		if newUser.Account.ID != oldUser.Account.ID {
			t.Errorf("Account is not equal when load with joins, loaded account id: %v, expect: %v", newUser.Account.ID, oldUser.Account.ID)
		}
	}

	t.Run("Joins", func(t *testing.T) {
		user := User{
			Name:    "joins-1",
			Company: Company{Name: "company"},
			Manager: &User{Name: "manager"},
			Account: Account{Number: "account-has-one-association"},
		}

		db.Create(&user)

		var user2 User
		if err := db.Joins("Company").Joins("Manager").Joins("Account").First(&user2, "users.name = ?", user.Name).Error; err != nil {
			t.Fatalf("Failed to load with joins, got error: %v", err)
		}

		check(t, user, user2)
	})

	t.Run("JoinsForSlice", func(t *testing.T) {
		users := []User{{
			Name:    "slice-joins-1",
			Company: Company{Name: "company"},
			Manager: &User{Name: "manager"},
			Account: Account{Number: "account-has-one-association"},
		}, {
			Name:    "slice-joins-2",
			Company: Company{Name: "company2"},
			Manager: &User{Name: "manager2"},
			Account: Account{Number: "account-has-one-association2"},
		}, {
			Name:    "slice-joins-3",
			Company: Company{Name: "company3"},
			Manager: &User{Name: "manager3"},
			Account: Account{Number: "account-has-one-association3"},
		}}

		db.Create(&users)

		var users2 []User
		if err := db.Joins("Company").Joins("Manager").Joins("Account").Find(&users2, "users.name LIKE ?", "slice-joins%").Error; err != nil {
			t.Fatalf("Failed to load with joins, got error: %v", err)
		} else if len(users2) != len(users) {
			t.Fatalf("Failed to load join users, got: %v, expect: %v", len(users2), len(users))
		}

		for _, u2 := range users2 {
			for _, u := range users {
				if u.Name == u2.Name {
					check(t, u, u2)
					continue
				}
			}
		}
	})
}
