package tests

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestAssociations(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&Account{}, &Company{}, &Pet{}, &Toy{}, &Language{})
	db.Migrator().AutoMigrate(&Account{}, &Company{}, &Pet{}, &Toy{}, &Language{})

	TestBelongsToAssociations(t, db)
}

func TestBelongsToAssociations(t *testing.T, db *gorm.DB) {
	check := func(t *testing.T, user User, old User) {
		if old.Company.Name != "" {
			if user.CompanyID == nil {
				t.Errorf("Company's foreign key should be saved")
			} else {
				var company Company
				db.First(&company, "id = ?", *user.CompanyID)
				if company.Name != old.Company.Name {
					t.Errorf("Company's name should be same, expects: %v, got %v", old.Company.Name, user.Company.Name)
				} else if user.Company.Name != old.Company.Name {
					t.Errorf("Company's name should be same, expects: %v, got %v", old.Company.Name, user.Company.Name)
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
		db.Find(&user2, "id = ?", user.ID)
		db.Model(&user2).Association("Company").Find(&user2.Company)
		user2.Manager = &User{}
		db.Model(&user2).Association("Manager").Find(user2.Manager)
		check(t, user2, user)
	})
}
