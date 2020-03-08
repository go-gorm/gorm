package tests

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func Now() *time.Time {
	now := time.Now()
	return &now
}

func RunTestsSuit(t *testing.T, db *gorm.DB) {
	TestCreate(t, db)
	TestFind(t, db)
	TestUpdate(t, db)
	TestDelete(t, db)

	TestGroupBy(t, db)
	TestJoins(t, db)
}

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
			t.Errorf("errors happened when create: %v", err)
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
	})
}

func TestFind(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	t.Run("Find", func(t *testing.T) {
		var users = []User{{
			Name:     "find",
			Age:      1,
			Birthday: Now(),
		}, {
			Name:     "find",
			Age:      2,
			Birthday: Now(),
		}, {
			Name:     "find",
			Age:      3,
			Birthday: Now(),
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Errorf("errors happened when create users: %v", err)
		}

		t.Run("First", func(t *testing.T) {
			var first User
			if err := db.Where("name = ?", "find").First(&first).Error; err != nil {
				t.Errorf("errors happened when query first: %v", err)
			} else {
				AssertObjEqual(t, first, users[0], "Name", "Age", "Birthday")
			}
		})

		t.Run("Last", func(t *testing.T) {
			var last User
			if err := db.Where("name = ?", "find").Last(&last).Error; err != nil {
				t.Errorf("errors happened when query last: %v", err)
			} else {
				AssertObjEqual(t, last, users[2], "Name", "Age", "Birthday")
			}
		})

		var all []User
		if err := db.Where("name = ?", "find").Find(&all).Error; err != nil || len(all) != 3 {
			t.Errorf("errors happened when query find: %v, length: %v", err, len(all))
		} else {
			for idx, user := range users {
				t.Run("FindAll#"+strconv.Itoa(idx+1), func(t *testing.T) {
					AssertObjEqual(t, all[idx], user, "Name", "Age", "Birthday")
				})
			}
		}

		t.Run("FirstMap", func(t *testing.T) {
			var first = map[string]interface{}{}
			if err := db.Model(&User{}).Where("name = ?", "find").First(first).Error; err != nil {
				t.Errorf("errors happened when query first: %v", err)
			} else {
				for _, name := range []string{"Name", "Age", "Birthday"} {
					t.Run(name, func(t *testing.T) {
						dbName := db.NamingStrategy.ColumnName("", name)
						reflectValue := reflect.Indirect(reflect.ValueOf(users[0]))
						AssertEqual(t, first[dbName], reflectValue.FieldByName(name).Interface())
					})
				}
			}
		})

		var allMap = []map[string]interface{}{}
		if err := db.Model(&User{}).Where("name = ?", "find").Find(&allMap).Error; err != nil {
			t.Errorf("errors happened when query first: %v", err)
		} else {
			for idx, user := range users {
				t.Run("FindAllMap#"+strconv.Itoa(idx+1), func(t *testing.T) {
					for _, name := range []string{"Name", "Age", "Birthday"} {
						t.Run(name, func(t *testing.T) {
							dbName := db.NamingStrategy.ColumnName("", name)
							reflectValue := reflect.Indirect(reflect.ValueOf(user))
							AssertEqual(t, allMap[idx][dbName], reflectValue.FieldByName(name).Interface())
						})
					}
				})
			}
		}
	})
}

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
			t.Errorf("errors happened when create: %v", err)
		} else if user.ID == 0 {
			t.Errorf("user's primary value should not zero, %v", user.ID)
		} else if user.UpdatedAt.IsZero() {
			t.Errorf("user's updated at should not zero, %v", user.UpdatedAt)
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

func TestDelete(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	t.Run("Delete", func(t *testing.T) {
		var users = []User{{
			Name:     "find",
			Age:      1,
			Birthday: Now(),
		}, {
			Name:     "find",
			Age:      2,
			Birthday: Now(),
		}, {
			Name:     "find",
			Age:      3,
			Birthday: Now(),
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Errorf("errors happened when create: %v", err)
		}

		for _, user := range users {
			if user.ID == 0 {
				t.Errorf("user's primary key should has value after create, got : %v", user.ID)
			}
		}

		if err := db.Delete(&users[1]).Error; err != nil {
			t.Errorf("errors happened when delete: %v", err)
		}

		var result User
		if err := db.Where("id = ?", users[1].ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("should returns record not found error, but got %v", err)
		}

		for _, user := range []User{users[0], users[2]} {
			if err := db.Where("id = ?", user.ID).First(&result).Error; err != nil {
				t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
			}
		}

		if err := db.Delete(&User{}).Error; err == nil || !errors.Is(err, gorm.ErrMissingWhereClause) {
			t.Errorf("should returns missing WHERE clause while deleting error")
		}

		for _, user := range []User{users[0], users[2]} {
			if err := db.Where("id = ?", user.ID).First(&result).Error; err != nil {
				t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
			}
		}
	})
}
