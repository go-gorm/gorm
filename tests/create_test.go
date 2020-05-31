package tests_test

import (
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func TestCreate(t *testing.T) {
	var user = *GetUser("create", Config{})

	if results := DB.Create(&user); results.Error != nil {
		t.Fatalf("errors happened when create: %v", results.Error)
	} else if results.RowsAffected != 1 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
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
	if err := DB.Where("id = ?", user.ID).First(&newUser).Error; err != nil {
		t.Fatalf("errors happened when query: %v", err)
	} else {
		CheckUser(t, newUser, user)
	}
}

func TestCreateWithAssociations(t *testing.T) {
	var user = *GetUser("create_with_associations", Config{
		Account:   true,
		Pets:      2,
		Toys:      3,
		Company:   true,
		Manager:   true,
		Team:      4,
		Languages: 3,
		Friends:   1,
	})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	CheckUser(t, user, user)

	var user2 User
	DB.Preload("Account").Preload("Pets").Preload("Toys").Preload("Company").Preload("Manager").Preload("Team").Preload("Languages").Preload("Friends").Find(&user2, "id = ?", user.ID)
	CheckUser(t, user2, user)
}

func TestBulkCreateWithAssociations(t *testing.T) {
	users := []User{
		*GetUser("bulk_1", Config{Account: true, Pets: 2, Toys: 3, Company: true, Manager: true, Team: 0, Languages: 1, Friends: 1}),
		*GetUser("bulk_2", Config{Account: false, Pets: 2, Toys: 4, Company: false, Manager: false, Team: 1, Languages: 3, Friends: 5}),
		*GetUser("bulk_3", Config{Account: true, Pets: 0, Toys: 3, Company: true, Manager: false, Team: 4, Languages: 0, Friends: 1}),
		*GetUser("bulk_4", Config{Account: true, Pets: 3, Toys: 0, Company: false, Manager: true, Team: 0, Languages: 3, Friends: 0}),
		*GetUser("bulk_5", Config{Account: false, Pets: 0, Toys: 3, Company: true, Manager: false, Team: 1, Languages: 3, Friends: 1}),
		*GetUser("bulk_6", Config{Account: true, Pets: 4, Toys: 3, Company: false, Manager: true, Team: 1, Languages: 3, Friends: 0}),
		*GetUser("bulk_7", Config{Account: true, Pets: 1, Toys: 3, Company: true, Manager: true, Team: 4, Languages: 3, Friends: 1}),
		*GetUser("bulk_8", Config{Account: false, Pets: 0, Toys: 0, Company: false, Manager: false, Team: 0, Languages: 0, Friends: 0}),
	}

	if results := DB.Create(&users); results.Error != nil {
		t.Fatalf("errors happened when create: %v", results.Error)
	} else if results.RowsAffected != int64(len(users)) {
		t.Fatalf("rows affected expects: %v, got %v", len(users), results.RowsAffected)
	}

	var userIDs []uint
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
		CheckUser(t, user, user)
	}

	var users2 []User
	DB.Preload("Account").Preload("Pets").Preload("Toys").Preload("Company").Preload("Manager").Preload("Team").Preload("Languages").Preload("Friends").Find(&users2, "id IN ?", userIDs)
	for idx, user := range users2 {
		CheckUser(t, user, users[idx])
	}
}

func TestBulkCreatePtrDataWithAssociations(t *testing.T) {
	users := []*User{
		GetUser("bulk_ptr_1", Config{Account: true, Pets: 2, Toys: 3, Company: true, Manager: true, Team: 0, Languages: 1, Friends: 1}),
		GetUser("bulk_ptr_2", Config{Account: false, Pets: 2, Toys: 4, Company: false, Manager: false, Team: 1, Languages: 3, Friends: 5}),
		GetUser("bulk_ptr_3", Config{Account: true, Pets: 0, Toys: 3, Company: true, Manager: false, Team: 4, Languages: 0, Friends: 1}),
		GetUser("bulk_ptr_4", Config{Account: true, Pets: 3, Toys: 0, Company: false, Manager: true, Team: 0, Languages: 3, Friends: 0}),
		GetUser("bulk_ptr_5", Config{Account: false, Pets: 0, Toys: 3, Company: true, Manager: false, Team: 1, Languages: 3, Friends: 1}),
		GetUser("bulk_ptr_6", Config{Account: true, Pets: 4, Toys: 3, Company: false, Manager: true, Team: 1, Languages: 3, Friends: 0}),
		GetUser("bulk_ptr_7", Config{Account: true, Pets: 1, Toys: 3, Company: true, Manager: true, Team: 4, Languages: 3, Friends: 1}),
		GetUser("bulk_ptr_8", Config{Account: false, Pets: 0, Toys: 0, Company: false, Manager: false, Team: 0, Languages: 0, Friends: 0}),
	}

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	var userIDs []uint
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
		CheckUser(t, *user, *user)
	}

	var users2 []User
	DB.Preload("Account").Preload("Pets").Preload("Toys").Preload("Company").Preload("Manager").Preload("Team").Preload("Languages").Preload("Friends").Find(&users2, "id IN ?", userIDs)
	for idx, user := range users2 {
		CheckUser(t, user, *users[idx])
	}
}

func TestPolymorphicHasOne(t *testing.T) {
	t.Run("Struct", func(t *testing.T) {
		var pet = Pet{
			Name: "PolymorphicHasOne",
			Toy:  Toy{Name: "Toy-PolymorphicHasOne"},
		}

		if err := DB.Create(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		CheckPet(t, pet, pet)

		var pet2 Pet
		DB.Preload("Toy").Find(&pet2, "id = ?", pet.ID)
		CheckPet(t, pet2, pet)
	})

	t.Run("Slice", func(t *testing.T) {
		var pets = []Pet{{
			Name: "PolymorphicHasOne-Slice-1",
			Toy:  Toy{Name: "Toy-PolymorphicHasOne-Slice-1"},
		}, {
			Name: "PolymorphicHasOne-Slice-2",
			Toy:  Toy{Name: "Toy-PolymorphicHasOne-Slice-2"},
		}, {
			Name: "PolymorphicHasOne-Slice-3",
			Toy:  Toy{Name: "Toy-PolymorphicHasOne-Slice-3"},
		}}

		if err := DB.Create(&pets).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var petIDs []uint
		for _, pet := range pets {
			petIDs = append(petIDs, pet.ID)
			CheckPet(t, pet, pet)
		}

		var pets2 []Pet
		DB.Preload("Toy").Find(&pets2, "id IN ?", petIDs)
		for idx, pet := range pets2 {
			CheckPet(t, pet, pets[idx])
		}
	})

	t.Run("SliceOfPtr", func(t *testing.T) {
		var pets = []*Pet{{
			Name: "PolymorphicHasOne-Slice-1",
			Toy:  Toy{Name: "Toy-PolymorphicHasOne-Slice-1"},
		}, {
			Name: "PolymorphicHasOne-Slice-2",
			Toy:  Toy{Name: "Toy-PolymorphicHasOne-Slice-2"},
		}, {
			Name: "PolymorphicHasOne-Slice-3",
			Toy:  Toy{Name: "Toy-PolymorphicHasOne-Slice-3"},
		}}

		if err := DB.Create(&pets).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		for _, pet := range pets {
			CheckPet(t, *pet, *pet)
		}
	})
}

func TestCreateEmptyStrut(t *testing.T) {
	type EmptyStruct struct {
		ID uint
	}
	DB.Migrator().DropTable(&EmptyStruct{})

	if err := DB.AutoMigrate(&EmptyStruct{}); err != nil {
		t.Errorf("no error should happen when auto migrate, but got %v", err)
	}

	if err := DB.Create(&EmptyStruct{}).Error; err != nil {
		t.Errorf("No error should happen when creating user, but got %v", err)
	}
}
