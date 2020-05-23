package tests_test

import (
	"testing"
	"time"

	. "github.com/jinzhu/gorm/tests"
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

	checkUpdatedTime := func(name string, n time.Time) {
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
	checkUpdatedTime("Update", user.UpdatedAt)
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
	checkUpdatedTime("Updates with map", user.UpdatedAt)
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
	checkUpdatedTime("Updates with struct", user.UpdatedAt)
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
	checkUpdatedTime("Save", user.UpdatedAt)
	checkOtherData("Save")

	var result4 User
	if err := DB.Where("id = ?", user.ID).First(&result4).Error; err != nil {
		t.Errorf("errors happened when query: %v", err)
	} else {
		CheckUser(t, result4, *user)
	}
}

func TestUpdateBelongsTo(t *testing.T) {
	var user = *GetUser("update-belongs-to", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Company = Company{Name: "company-belongs-to-association"}
	user.Manager = &User{Name: "manager-belongs-to-association"}
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Company").Preload("Manager").Find(&user2, "id = ?", user.ID)
	CheckUser(t, user2, user)
}

func TestUpdateHasOne(t *testing.T) {
	var user = *GetUser("update-has-one", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Account = Account{Number: "account-has-one-association"}

	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Account").Find(&user2, "id = ?", user.ID)
	CheckUser(t, user2, user)

	t.Run("Polymorphic", func(t *testing.T) {
		var pet = Pet{Name: "create"}

		if err := DB.Create(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		pet.Toy = Toy{Name: "Update-HasOneAssociation-Polymorphic"}

		if err := DB.Save(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var pet2 Pet
		DB.Preload("Toy").Find(&pet2, "id = ?", pet.ID)
		CheckPet(t, pet2, pet)
	})
}

func TestUpdateHasManyAssociations(t *testing.T) {
	var user = *GetUser("update-has-many", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Pets = []*Pet{{Name: "pet1"}, {Name: "pet2"}}
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Pets").Find(&user2, "id = ?", user.ID)
	CheckUser(t, user2, user)

	t.Run("Polymorphic", func(t *testing.T) {
		var user = *GetUser("update-has-many", Config{})

		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		user.Toys = []Toy{{Name: "toy1"}, {Name: "toy2"}}
		if err := DB.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		var user2 User
		DB.Preload("Toys").Find(&user2, "id = ?", user.ID)
		CheckUser(t, user2, user)
	})
}

func TestUpdateMany2ManyAssociations(t *testing.T) {
	var user = *GetUser("update-many2many", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Languages = []Language{{Code: "zh-CN", Name: "Chinese"}, {Code: "en", Name: "English"}}
	for _, lang := range user.Languages {
		DB.Create(&lang)
	}
	user.Friends = []*User{{Name: "friend-1"}, {Name: "friend-2"}}

	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Languages").Preload("Friends").Find(&user2, "id = ?", user.ID)
	CheckUser(t, user2, user)
}
