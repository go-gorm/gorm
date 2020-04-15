package tests

import (
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
	})
}
