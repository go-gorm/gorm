package tests_test

import (
	"testing"
	"time"

	"gorm.io/gorm/clause"
	. "gorm.io/gorm/utils/tests"
)

func TestUpsert(t *testing.T) {
	lang := Language{Code: "upsert", Name: "Upsert"}
	DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&lang)

	lang2 := Language{Code: "upsert", Name: "Upsert"}
	DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&lang2)

	var langs []Language
	if err := DB.Find(&langs, "code = ?", lang.Code).Error; err != nil {
		t.Errorf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs) != 1 {
		t.Errorf("should only find only 1 languages, but got %+v", langs)
	}
}

func TestUpsertSlice(t *testing.T) {
	langs := []Language{
		{Code: "upsert-slice1", Name: "Upsert-slice1"},
		{Code: "upsert-slice2", Name: "Upsert-slice2"},
		{Code: "upsert-slice3", Name: "Upsert-slice3"},
	}
	DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&langs)

	var langs2 []Language
	if err := DB.Find(&langs2, "code LIKE ?", "upsert-slice%").Error; err != nil {
		t.Errorf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs2) != 3 {
		t.Errorf("should only find only 3 languages, but got %+v", langs2)
	}

	DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&langs)
	var langs3 []Language
	if err := DB.Find(&langs3, "code LIKE ?", "upsert-slice%").Error; err != nil {
		t.Errorf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs3) != 3 {
		t.Errorf("should only find only 3 languages, but got %+v", langs3)
	}
}

func TestFindOrInitialize(t *testing.T) {
	var user1, user2, user3, user4, user5, user6 User
	if err := DB.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user1).Error; err != nil {
		t.Errorf("no error should happen when FirstOrInit, but got %v", err)
	}

	if user1.Name != "find or init" || user1.ID != 0 || user1.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	DB.Where(User{Name: "find or init", Age: 33}).FirstOrInit(&user2)
	if user2.Name != "find or init" || user2.ID != 0 || user2.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	DB.FirstOrInit(&user3, map[string]interface{}{"name": "find or init 2"})
	if user3.Name != "find or init 2" || user3.ID != 0 {
		t.Errorf("user should be initialized with inline search value")
	}

	DB.Where(&User{Name: "find or init"}).Attrs(User{Age: 44}).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.ID != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and attrs")
	}

	DB.Where(&User{Name: "find or init"}).Assign("age", 44).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.ID != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and assign attrs")
	}

	DB.Save(&User{Name: "find or init", Age: 33})
	DB.Where(&User{Name: "find or init"}).Attrs("age", 44).FirstOrInit(&user5)
	if user5.Name != "find or init" || user5.ID == 0 || user5.Age != 33 {
		t.Errorf("user should be found and not initialized by Attrs")
	}

	DB.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.ID == 0 || user6.Age != 33 {
		t.Errorf("user should be found with FirstOrInit")
	}

	DB.Where(&User{Name: "find or init"}).Assign(User{Age: 44}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.ID == 0 || user6.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}
}

func TestFindOrCreate(t *testing.T) {
	var user1, user2, user3, user4, user5, user6, user7, user8 User
	if err := DB.Where(&User{Name: "find or create", Age: 33}).FirstOrCreate(&user1).Error; err != nil {
		t.Errorf("no error should happen when FirstOrInit, but got %v", err)
	}

	if user1.Name != "find or create" || user1.ID == 0 || user1.Age != 33 {
		t.Errorf("user should be created with search value")
	}

	DB.Where(&User{Name: "find or create", Age: 33}).FirstOrCreate(&user2)
	if user1.ID != user2.ID || user2.Name != "find or create" || user2.ID == 0 || user2.Age != 33 {
		t.Errorf("user should be created with search value")
	}

	DB.FirstOrCreate(&user3, map[string]interface{}{"name": "find or create 2"})
	if user3.Name != "find or create 2" || user3.ID == 0 {
		t.Errorf("user should be created with inline search value")
	}

	DB.Where(&User{Name: "find or create 3"}).Attrs("age", 44).FirstOrCreate(&user4)
	if user4.Name != "find or create 3" || user4.ID == 0 || user4.Age != 44 {
		t.Errorf("user should be created with search value and attrs")
	}

	updatedAt1 := user4.UpdatedAt
	DB.Where(&User{Name: "find or create 3"}).Assign("age", 55).FirstOrCreate(&user4)

	if user4.Age != 55 {
		t.Errorf("Failed to set change to 55, got %v", user4.Age)
	}

	if updatedAt1.Format(time.RFC3339Nano) == user4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("UpdateAt should be changed when update values with assign")
	}

	DB.Where(&User{Name: "find or create 4"}).Assign(User{Age: 44}).FirstOrCreate(&user4)
	if user4.Name != "find or create 4" || user4.ID == 0 || user4.Age != 44 {
		t.Errorf("user should be created with search value and assigned attrs")
	}

	DB.Where(&User{Name: "find or create"}).Attrs("age", 44).FirstOrInit(&user5)
	if user5.Name != "find or create" || user5.ID == 0 || user5.Age != 33 {
		t.Errorf("user should be found and not initialized by Attrs")
	}

	DB.Where(&User{Name: "find or create"}).Assign(User{Age: 44}).FirstOrCreate(&user6)
	if user6.Name != "find or create" || user6.ID == 0 || user6.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}

	DB.Where(&User{Name: "find or create"}).Find(&user7)
	if user7.Name != "find or create" || user7.ID == 0 || user7.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}

	DB.Where(&User{Name: "find or create embedded struct"}).Assign(User{Age: 44, Account: Account{Number: "1231231231"}, Pets: []*Pet{{Name: "first_or_create_pet1"}, {Name: "first_or_create_pet2"}}}).FirstOrCreate(&user8)
	if DB.Where("name = ?", "first_or_create_pet1").First(&Pet{}).RecordNotFound() {
		t.Errorf("has many association should be saved")
	}

	if DB.Where("number = ?", "1231231231").First(&Account{}).RecordNotFound() {
		t.Errorf("belongs to association should be saved")
	}
}
