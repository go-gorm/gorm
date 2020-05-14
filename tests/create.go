package tests

import (
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestCreate(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	t.Run("Create", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		if user.ID == 0 {
			t.Errorf("user's primary key should has value after create, got : %v", user.ID)
		}

		if user.CreatedAt.IsZero() {
			t.Errorf("user's created at should be not zero")
		}

		if user.UpdatedAt.IsZero() {
			t.Errorf("user's updated at should be not zero")
		}

		var newUser User
		if err := db.Where("id = ?", user.ID).First(&newUser).Error; err != nil {
			t.Errorf("errors happened when query: %v", err)
		} else {
			AssertObjEqual(t, newUser, user, "Name", "Age", "Birthday")
		}

		TestCreateAssociations(t, db)
	})
}

func TestCreateAssociations(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&Account{}, &Company{}, &Pet{}, &Toy{}, &Language{})
	db.Migrator().AutoMigrate(&Account{}, &Company{}, &Pet{}, &Toy{}, &Language{})

	TestCreateBelongsToAssociations(t, db)
	TestCreateHasOneAssociations(t, db)
	TestCreateHasManyAssociations(t, db)
	TestCreateMany2ManyAssociations(t, db)
}

func TestCreateBelongsToAssociations(t *testing.T, db *gorm.DB) {
	check := func(t *testing.T, user User, old User) {
		if old.Company.Name != "" {
			if user.CompanyID == nil {
				t.Errorf("Company's foreign key should be saved")
			} else {
				var company Company
				db.First(&company, "id = ?", *user.CompanyID)
				if company.Name != old.Company.Name {
					t.Errorf("Company's name should be same")
				} else if user.Company.Name != old.Company.Name {
					t.Errorf("Company's name should be same")
				}
			}
		} else if user.CompanyID != nil {
			t.Errorf("Company should not be created for zero value, got: %+v", user.CompanyID)
		}

		if old.Manager != nil {
			if user.ManagerID == nil {
				t.Errorf("Manager's foreign key should be saved")
			} else {
				var manager User
				db.First(&manager, "id = ?", *user.ManagerID)
				if manager.Name != user.Manager.Name {
					t.Errorf("Manager's name should be same")
				} else if user.Manager.Name != old.Manager.Name {
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
			Company:  Company{Name: "company-belongs-to-association"},
			Manager:  &User{Name: "manager-belongs-to-association"},
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		check(t, user, user)

		var user2 User
		db.Preload("Company").Preload("Manager").Find(&user2, "id = ?", user.ID)
		check(t, user2, user)
	})

	t.Run("BelongsToForBulkInsert", func(t *testing.T) {
		var users = []User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association-1"},
			Manager:  &User{Name: "manager-belongs-to-association-1"},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association-2"},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association-3"},
			Manager:  &User{Name: "manager-belongs-to-association-3"},
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var userIDs []uint
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
			check(t, user, user)
		}

		var users2 []User
		db.Preload("Company").Preload("Manager").Find(&users2, "id IN (?)", userIDs)
		for idx, user := range users2 {
			check(t, user, users[idx])
		}

		var users3 []User
		db.Preload("Company").Preload("Manager").Find(users3, "id IN (?)", userIDs)
		for idx, user := range users3 {
			check(t, user, users[idx])
		}
	})

	t.Run("BelongsToForBulkInsertPtrData", func(t *testing.T) {
		var users = []*User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association-1"},
			Manager:  &User{Name: "manager-belongs-to-association-1"},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association-2"},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association-3"},
			Manager:  &User{Name: "manager-belongs-to-association-3"},
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, user := range users {
			check(t, *user, *user)
		}
	})

	t.Run("BelongsToForBulkInsertWithoutPtr", func(t *testing.T) {
		var users = []*User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association-1"},
			Manager:  &User{Name: "manager-belongs-to-association-1"},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association-2"},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association-3"},
			Manager:  &User{Name: "manager-belongs-to-association-3"},
		}}

		if err := db.Create(users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, user := range users {
			check(t, *user, *user)
		}
	})
}

func TestCreateHasOneAssociations(t *testing.T, db *gorm.DB) {
	check := func(t *testing.T, user User, old User) {
		if user.Account.ID == 0 {
			t.Errorf("Account should be saved")
		} else if user.Account.UserID.Int64 != int64(user.ID) {
			t.Errorf("Account's foreign key should be saved")
		} else {
			var account Account
			db.First(&account, "id = ?", user.Account.ID)
			if account.Number != user.Account.Number {
				t.Errorf("Account's number should be same")
			} else if user.Account.Number != old.Account.Number {
				t.Errorf("Account's number should be same")
			}
		}
	}

	t.Run("HasOne", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association"},
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		check(t, user, user)

		var user2 User
		db.Preload("Account").Find(&user2, "id = ?", user.ID)
		check(t, user2, user)
	})

	t.Run("HasOneForBulkInsert", func(t *testing.T) {
		var users = []User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association-1"},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association-2"},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association-3"},
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var userIDs []uint
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
			check(t, user, user)
		}

		var users2 []User
		db.Preload("Account").Find(&users2, "id IN (?)", userIDs)
		for idx, user := range users2 {
			check(t, user, users[idx])
		}
	})

	t.Run("HasOneForBulkInsertPtrData", func(t *testing.T) {
		var users = []*User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association-1"},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association-2"},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association-3"},
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, user := range users {
			check(t, *user, *user)
		}
	})

	t.Run("HasOneForBulkInsertWithoutPtr", func(t *testing.T) {
		var users = []User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association-1"},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association-2"},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association-3"},
		}}

		if err := db.Create(users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, user := range users {
			check(t, user, user)
		}
	})

	checkPet := func(t *testing.T, pet Pet, old Pet) {
		if pet.Toy.OwnerID != fmt.Sprint(pet.ID) || pet.Toy.OwnerType != "pets" {
			t.Errorf("Failed to create polymorphic has one association - toy owner id %v, owner type %v", pet.Toy.OwnerID, pet.Toy.OwnerType)
		} else {
			var toy Toy
			db.First(&toy, "owner_id = ? and owner_type = ?", pet.Toy.OwnerID, pet.Toy.OwnerType)
			if toy.Name != pet.Toy.Name {
				t.Errorf("Failed to query saved polymorphic has one association")
			} else if old.Toy.Name != pet.Toy.Name {
				t.Errorf("Failed to query saved polymorphic has one association")
			}
		}
	}

	t.Run("PolymorphicHasOne", func(t *testing.T) {
		var pet = Pet{
			Name: "create",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic"},
		}

		if err := db.Create(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		checkPet(t, pet, pet)

		var pet2 Pet
		db.Preload("Toy").Find(&pet2, "id = ?", pet.ID)
		checkPet(t, pet2, pet)
	})

	t.Run("PolymorphicHasOneForBulkInsert", func(t *testing.T) {
		var pets = []Pet{{
			Name: "create-1",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic-1"},
		}, {
			Name: "create-2",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic-2"},
		}, {
			Name: "create-3",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic-3"},
		}}

		if err := db.Create(&pets).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var petIDs []uint
		for _, pet := range pets {
			petIDs = append(petIDs, pet.ID)
			checkPet(t, pet, pet)
		}

		var pets2 []Pet
		db.Preload("Toy").Find(&pets2, "id IN (?)", petIDs)
		for idx, pet := range pets2 {
			checkPet(t, pet, pets[idx])
		}
	})

	t.Run("PolymorphicHasOneForBulkInsertPtrData", func(t *testing.T) {
		var pets = []*Pet{{
			Name: "create-1",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic-1"},
		}, {
			Name: "create-2",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic-2"},
		}, {
			Name: "create-3",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic-3"},
		}}

		if err := db.Create(&pets).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, pet := range pets {
			checkPet(t, *pet, *pet)
		}
	})

	t.Run("PolymorphicHasOneForBulkInsertWithoutPtr", func(t *testing.T) {
		var pets = []*Pet{{
			Name: "create-1",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic-1"},
		}, {
			Name: "create-2",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic-2"},
		}, {
			Name: "create-3",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic-3"},
		}}

		if err := db.Create(pets).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, pet := range pets {
			checkPet(t, *pet, *pet)
		}
	})
}

func TestCreateHasManyAssociations(t *testing.T, db *gorm.DB) {
	check := func(t *testing.T, user User, old User) {
		for idx, pet := range user.Pets {
			if pet.ID == 0 {
				t.Errorf("Pet's foreign key should be saved")
			}

			var result Pet
			db.First(&result, "id = ?", pet.ID)
			if result.Name != pet.Name {
				t.Errorf("Pet's name should be same")
			} else if result.UserID != user.ID {
				t.Errorf("Pet's foreign key should be saved")
			} else if result.Name != old.Pets[idx].Name {
				t.Errorf("Pet's name should be same")
			}
		}
	}

	t.Run("HasMany", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet1"}, {Name: "pet2"}},
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		check(t, user, user)

		var user2 User
		db.Preload("Pets").Find(&user2, "id = ?", user.ID)
		check(t, user2, user)
	})

	t.Run("HasManyForBulkInsert", func(t *testing.T) {
		var users = []User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet-1-1"}, {Name: "pet-1-2"}},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet-2-1"}},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet-3-1"}, {Name: "pet-3-2"}},
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var userIDs []uint
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
			check(t, user, user)
		}

		var users2 []User
		db.Preload("Pets").Find(&users2, "id IN (?)", userIDs)
		for idx, user := range users2 {
			check(t, user, users[idx])
		}
	})

	t.Run("HasManyForBulkInsertPtrData", func(t *testing.T) {
		var users = []*User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet-1-1"}, {Name: "pet-1-2"}},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet-2-1"}},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet-3-1"}, {Name: "pet-3-2"}},
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, user := range users {
			check(t, *user, *user)
		}
	})

	t.Run("HasManyForBulkInsertWithoutPtr", func(t *testing.T) {
		var users = []*User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet-1-1"}, {Name: "pet-1-2"}},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet-2-1"}},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Pets:     []*Pet{{Name: "pet-3-1"}, {Name: "pet-3-2"}},
		}}

		if err := db.Create(users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, user := range users {
			check(t, *user, *user)
		}
	})

	checkToy := func(t *testing.T, user User, old User) {
		for idx, toy := range user.Toys {
			if toy.ID == 0 {
				t.Fatalf("Failed to create toy #%v", idx)
			}

			var result Toy
			db.First(&result, "id = ?", toy.ID)
			if result.Name != toy.Name {
				t.Errorf("Failed to query saved toy")
			} else if result.Name != old.Toys[idx].Name {
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
			Toys:     []Toy{{Name: "toy1"}, {Name: "toy2"}},
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		checkToy(t, user, user)

		var user2 User
		db.Preload("Toys").Find(&user2, "id = ?", user.ID)
		check(t, user2, user)
	})

	t.Run("PolymorphicHasManyForBulkInsert", func(t *testing.T) {
		var users = []User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Toys:     []Toy{{Name: "toy-1-1"}, {Name: "toy-1-2"}},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Toys:     []Toy{{Name: "toy-2-1"}, {Name: "toy-2-2"}},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Toys:     []Toy{{Name: "toy-3-1"}, {Name: "toy-3-2"}},
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var userIDs []uint
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
			checkToy(t, user, user)
		}

		var users2 []User
		db.Preload("Toys").Find(&users2, "id IN (?)", userIDs)
		for idx, user := range users2 {
			check(t, user, users[idx])
		}
	})

	t.Run("PolymorphicHasManyForBulkInsertPtrData", func(t *testing.T) {
		var users = []*User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Toys:     []Toy{{Name: "toy-1-1"}, {Name: "toy-1-2"}},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Toys:     []Toy{{Name: "toy-2-1"}, {Name: "toy-2-2"}},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Toys:     []Toy{{Name: "toy-3-1"}, {Name: "toy-3-2"}},
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, user := range users {
			checkToy(t, *user, *user)
		}
	})

	t.Run("PolymorphicHasManyForBulkInsertWithoutPtr", func(t *testing.T) {
		var users = []User{{
			Name:     "create-1",
			Age:      18,
			Birthday: Now(),
			Toys:     []Toy{{Name: "toy-1-1"}, {Name: "toy-1-2"}},
		}, {
			Name:     "create-2",
			Age:      28,
			Birthday: Now(),
			Toys:     []Toy{{Name: "toy-2-1"}, {Name: "toy-2-2"}},
		}, {
			Name:     "create-3",
			Age:      38,
			Birthday: Now(),
			Toys:     []Toy{{Name: "toy-3-1"}, {Name: "toy-3-2"}},
		}}

		if err := db.Create(users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, user := range users {
			checkToy(t, user, user)
		}
	})
}

func TestCreateMany2ManyAssociations(t *testing.T, db *gorm.DB) {
	check := func(t *testing.T, user User, old User) {
		for idx, language := range user.Languages {
			var result Language
			db.First(&result, "code = ?", language.Code)
			if result.Name != language.Name {
				t.Errorf("Language's name should be same")
			} else if result.Name != old.Languages[idx].Name {
				t.Errorf("Language's name should be same")
			}
		}

		for idx, f := range user.Friends {
			if f.ID == 0 {
				t.Errorf("Friend's foreign key should be saved")
			}

			var result User
			db.First(&result, "id = ?", f.ID)
			if result.Name != f.Name {
				t.Errorf("Friend's name should be same")
			} else if result.Name != old.Friends[idx].Name {
				t.Errorf("Language's name should be same")
			}
		}
	}

	db.Create(&[]Language{{Code: "zh-CN", Name: "Chinese"}, {Code: "en", Name: "English"}})

	t.Run("Many2Many", func(t *testing.T) {
		var user = User{
			Name:      "create",
			Age:       18,
			Birthday:  Now(),
			Languages: []Language{{Code: "zh-CN", Name: "Chinese"}, {Code: "en", Name: "English"}},
			Friends:   []*User{{Name: "friend-1"}, {Name: "friend-2"}},
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		check(t, user, user)

		var user2 User
		db.Preload("Languages").Preload("Friends").Find(&user2, "id = ?", user.ID)
		check(t, user2, user)
	})

	t.Run("Many2ManyForBulkInsert", func(t *testing.T) {
		var users = []User{
			{
				Name:      "create-1",
				Age:       18,
				Birthday:  Now(),
				Languages: []Language{{Code: "zh-CN", Name: "Chinese"}, {Code: "en", Name: "English"}},
				Friends:   []*User{{Name: "friend-1-1"}, {Name: "friend-1-2"}},
			},
			{
				Name:      "create-2",
				Age:       18,
				Birthday:  Now(),
				Languages: []Language{{Code: "zh-CN", Name: "Chinese"}},
				Friends:   []*User{{Name: "friend-2-1"}},
			},
			{
				Name:      "create-3",
				Age:       18,
				Birthday:  Now(),
				Languages: []Language{{Code: "zh-CN", Name: "Chinese"}, {Code: "en", Name: "English"}},
				Friends:   []*User{{Name: "friend-3-1"}, {Name: "friend-3-2"}, {Name: "friend-3-3"}},
			},
		}

		if err := db.Create(&users).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var userIDs []uint
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
			check(t, user, user)
		}

		var users2 []User
		db.Preload("Languages").Preload("Friends").Find(&users2, "id IN (?)", userIDs)
		for idx, user := range users2 {
			check(t, user, users[idx])
		}
	})
}
