package gorm_test

import "testing"

func TestOrderSQLInjection(t *testing.T) {
	DB.AutoMigrate(new(User))

	testUser := &User{Name: "jinzhu"}
	DB.Save(testUser)

	var countBefore int
	DB.Model(new(User)).Count(&countBefore)

	var users []*User
	DB.Order("id;delete from users;commit;").Find(&users)

	var countAfter int
	DB.Model(new(User)).Count(&countAfter)

	if countAfter != countBefore {
		t.Error("Seems like it's possible to use SQL injection with ORDER BY")
	}
}

func TestGroupSQLInjection(t *testing.T) {
	DB.AutoMigrate(new(User))

	testUser := &User{Name: "jinzhu"}
	DB.Save(testUser)

	var countBefore int
	DB.Model(new(User)).Count(&countBefore)

	var users []*User
	DB.Group("name;delete from users;commit;").Find(&users)

	var countAfter int
	DB.Model(new(User)).Count(&countAfter)

	if countAfter != countBefore {
		t.Error("Seems like it's possible to use SQL injection with GROUP BY")
	}
}
