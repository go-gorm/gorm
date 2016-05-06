package gorm_test

import "testing"

func TestOrderSQLInjection(t *testing.T) {
	DB.AutoMigrate(new(User))

	DB.Save(&User{Name: "jinzhu"})

	var users []*User
	DB.Order("id;delete from users;commit;").Find(&users)

	if len(users) != 1 {
		t.Error("Seems like it's possible to use SQL injection with ORDER BY")
	}
}

func TestGroupSQLInjection(t *testing.T) {
	DB.AutoMigrate(new(User))

	DB.Save(&User{Name: "jinzhu"})

	var users []*User
	DB.Group("name;delete from users;commit;").Find(&users)

	if len(users) != 1 {
		t.Error("Seems like it's possible to use SQL injection with GROUP BY")
	}
}
