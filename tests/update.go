package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestUpdate(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	t.Run("Update", func(t *testing.T) {
		var (
			users = []*User{{
				Name:     "update-before",
				Age:      1,
				Birthday: Now(),
			}, {
				Name:     "update",
				Age:      18,
				Birthday: Now(),
			}, {
				Name:     "update-after",
				Age:      1,
				Birthday: Now(),
			}}
			user          = users[1]
			lastUpdatedAt time.Time
		)

		checkUpdatedTime := func(name string, n time.Time) {
			if n.UnixNano() == lastUpdatedAt.UnixNano() {
				t.Errorf("%v: user's updated at should be changed, but got %v, was %v", name, n, lastUpdatedAt)
			}
			lastUpdatedAt = n
		}

		checkOtherData := func(name string) {
			var beforeUser, afterUser User
			if err := db.Where("id = ?", users[0].ID).First(&beforeUser).Error; err != nil {
				t.Errorf("errors happened when query before user: %v", err)
			}
			t.Run(name, func(t *testing.T) {
				AssertObjEqual(t, beforeUser, users[0], "Name", "Age", "Birthday")
			})

			if err := db.Where("id = ?", users[2].ID).First(&afterUser).Error; err != nil {
				t.Errorf("errors happened when query after user: %v", err)
			}
			t.Run(name, func(t *testing.T) {
				AssertObjEqual(t, afterUser, users[2], "Name", "Age", "Birthday")
			})
		}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		} else if user.ID == 0 {
			t.Fatalf("user's primary value should not zero, %v", user.ID)
		} else if user.UpdatedAt.IsZero() {
			t.Fatalf("user's updated at should not zero, %v", user.UpdatedAt)
		}
		lastUpdatedAt = user.UpdatedAt

		if err := db.Model(user).Update("Age", 10).Error; err != nil {
			t.Errorf("errors happened when update: %v", err)
		} else if user.Age != 10 {
			t.Errorf("Age should equals to 10, but got %v", user.Age)
		}
		checkUpdatedTime("Update", user.UpdatedAt)
		checkOtherData("Update")

		var result User
		if err := db.Where("id = ?", user.ID).First(&result).Error; err != nil {
			t.Errorf("errors happened when query: %v", err)
		} else {
			AssertObjEqual(t, result, user, "Name", "Age", "Birthday")
		}

		values := map[string]interface{}{"Active": true, "age": 5}
		if err := db.Model(user).Updates(values).Error; err != nil {
			t.Errorf("errors happened when update: %v", err)
		} else if user.Age != 5 {
			t.Errorf("Age should equals to 5, but got %v", user.Age)
		} else if user.Active != true {
			t.Errorf("Active should be true, but got %v", user.Active)
		}
		checkUpdatedTime("Updates with map", user.UpdatedAt)
		checkOtherData("Updates with map")

		var result2 User
		if err := db.Where("id = ?", user.ID).First(&result2).Error; err != nil {
			t.Errorf("errors happened when query: %v", err)
		} else {
			AssertObjEqual(t, result2, user, "Name", "Age", "Birthday")
		}

		if err := db.Model(user).Updates(User{Age: 2}).Error; err != nil {
			t.Errorf("errors happened when update: %v", err)
		} else if user.Age != 2 {
			t.Errorf("Age should equals to 2, but got %v", user.Age)
		}
		checkUpdatedTime("Updates with struct", user.UpdatedAt)
		checkOtherData("Updates with struct")

		var result3 User
		if err := db.Where("id = ?", user.ID).First(&result3).Error; err != nil {
			t.Errorf("errors happened when query: %v", err)
		} else {
			AssertObjEqual(t, result3, user, "Name", "Age", "Birthday")
		}

		user.Active = false
		user.Age = 1
		if err := db.Save(user).Error; err != nil {
			t.Errorf("errors happened when update: %v", err)
		} else if user.Age != 1 {
			t.Errorf("Age should equals to 1, but got %v", user.Age)
		} else if user.Active != false {
			t.Errorf("Active should equals to false, but got %v", user.Active)
		}
		checkUpdatedTime("Save", user.UpdatedAt)
		checkOtherData("Save")

		var result4 User
		if err := db.Where("id = ?", user.ID).First(&result4).Error; err != nil {
			t.Errorf("errors happened when query: %v", err)
		} else {
			AssertObjEqual(t, result4, user, "Name", "Age", "Birthday")
		}

		TestUpdateAssociations(t, db)
	})
}

func TestUpdateAssociations(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&Account{}, &Company{}, &Pet{}, &Toy{}, &Language{})
	db.Migrator().AutoMigrate(&Account{}, &Company{}, &Pet{}, &Toy{}, &Language{})

	TestUpdateBelongsToAssociations(t, db)
	TestUpdateHasOneAssociations(t, db)
	TestUpdateHasManyAssociations(t, db)
	TestUpdateMany2ManyAssociations(t, db)
}

func TestUpdateBelongsToAssociations(t *testing.T, db *gorm.DB) {
	check := func(t *testing.T, user User) {
		if user.Company.Name != "" {
			if user.CompanyID == nil {
				t.Errorf("Company's foreign key should be saved")
			} else {
				var company Company
				db.First(&company, "id = ?", *user.CompanyID)
				if company.Name != user.Company.Name {
					t.Errorf("Company's name should be same")
				}
			}
		} else if user.CompanyID != nil {
			t.Errorf("Company should not be created for zero value, got: %+v", user.CompanyID)
		}

		if user.Manager != nil {
			if user.ManagerID == nil {
				t.Errorf("Manager's foreign key should be saved")
			} else {
				var manager User
				db.First(&manager, "id = ?", *user.ManagerID)
				if manager.Name != user.Manager.Name {
					t.Errorf("Manager's name should be same")
				}
			}
		} else if user.ManagerID != nil {
			t.Errorf("Manager should not be created for zero value, got: %+v", user.ManagerID)
		}
	}

	t.Run("BelongsTo", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		user.Company = Company{Name: "company-belongs-to-association"}
		user.Manager = &User{Name: "manager-belongs-to-association"}
		if err := db.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		check(t, user)
	})
}

func TestUpdateHasOneAssociations(t *testing.T, db *gorm.DB) {
	check := func(t *testing.T, user User) {
		if user.Account.ID == 0 {
			t.Errorf("Account should be saved")
		} else if user.Account.UserID.Int64 != int64(user.ID) {
			t.Errorf("Account's foreign key should be saved")
		} else {
			var account Account
			db.First(&account, "id = ?", user.Account.ID)
			if account.Number != user.Account.Number {
				t.Errorf("Account's number should be sme")
			}
		}
	}

	t.Run("HasOne", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		user.Account = Account{Number: "account-has-one-association"}

		if err := db.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		check(t, user)
	})

	checkPet := func(t *testing.T, pet Pet) {
		if pet.Toy.OwnerID != fmt.Sprint(pet.ID) || pet.Toy.OwnerType != "pets" {
			t.Errorf("Failed to create polymorphic has one association - toy owner id %v, owner type %v", pet.Toy.OwnerID, pet.Toy.OwnerType)
		} else {
			var toy Toy
			db.First(&toy, "owner_id = ? and owner_type = ?", pet.Toy.OwnerID, pet.Toy.OwnerType)
			if toy.Name != pet.Toy.Name {
				t.Errorf("Failed to query saved polymorphic has one association")
			}
		}
	}

	t.Run("PolymorphicHasOne", func(t *testing.T) {
		var pet = Pet{
			Name: "create",
		}

		if err := db.Create(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		pet.Toy = Toy{Name: "Update-HasOneAssociation-Polymorphic"}

		if err := db.Save(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		checkPet(t, pet)
	})
}

func TestUpdateHasManyAssociations(t *testing.T, db *gorm.DB) {
	check := func(t *testing.T, user User) {
		for _, pet := range user.Pets {
			if pet.ID == 0 {
				t.Errorf("Pet's foreign key should be saved")
			}

			var result Pet
			db.First(&result, "id = ?", pet.ID)
			if result.Name != pet.Name {
				t.Errorf("Pet's name should be same")
			} else if result.UserID != user.ID {
				t.Errorf("Pet's foreign key should be saved")
			}
		}
	}

	t.Run("HasMany", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		user.Pets = []*Pet{{Name: "pet1"}, {Name: "pet2"}}
		if err := db.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		check(t, user)
	})

	checkToy := func(t *testing.T, user User) {
		for idx, toy := range user.Toys {
			if toy.ID == 0 {
				t.Fatalf("Failed to create toy #%v", idx)
			}

			var result Toy
			db.First(&result, "id = ?", toy.ID)
			if result.Name != toy.Name {
				t.Errorf("Failed to query saved toy")
			} else if result.OwnerID != fmt.Sprint(user.ID) || result.OwnerType != "users" {
				t.Errorf("Failed to save relation")
			}
		}
	}

	t.Run("PolymorphicHasMany", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		user.Toys = []Toy{{Name: "toy1"}, {Name: "toy2"}}
		if err := db.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		checkToy(t, user)
	})
}

func TestUpdateMany2ManyAssociations(t *testing.T, db *gorm.DB) {
	check := func(t *testing.T, user User) {
		for _, language := range user.Languages {
			var result Language
			db.First(&result, "code = ?", language.Code)
			// TODO
			// if result.Name != language.Name {
			// 	t.Errorf("Language's name should be same")
			// }
		}

		for _, f := range user.Friends {
			if f.ID == 0 {
				t.Errorf("Friend's foreign key should be saved")
			}

			var result User
			db.First(&result, "id = ?", f.ID)
			if result.Name != f.Name {
				t.Errorf("Friend's name should be same")
			}
		}
	}

	t.Run("Many2Many", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		user.Languages = []Language{{Code: "zh-CN", Name: "Chinese"}, {Code: "en", Name: "English"}}
		user.Friends = []*User{{Name: "friend-1"}, {Name: "friend-2"}}

		if err := db.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		check(t, user)
	})
}
