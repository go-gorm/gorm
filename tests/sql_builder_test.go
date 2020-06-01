package tests_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	. "github.com/jinzhu/gorm/tests"
)

func TestRow(t *testing.T) {
	user1 := User{Name: "RowUser1", Age: 1}
	user2 := User{Name: "RowUser2", Age: 10}
	user3 := User{Name: "RowUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	row := DB.Table("users").Where("name = ?", user2.Name).Select("age").Row()

	var age int64
	if err := row.Scan(&age); err != nil {
		t.Fatalf("Failed to scan age, got %v", err)
	}

	if age != 10 {
		t.Errorf("Scan with Row, age expects: %v, got %v", user2.Age, age)
	}
}

func TestRows(t *testing.T) {
	user1 := User{Name: "RowsUser1", Age: 1}
	user2 := User{Name: "RowsUser2", Age: 10}
	user3 := User{Name: "RowsUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	rows, err := DB.Table("users").Where("name = ? or name = ?", user2.Name, user3.Name).Select("name, age").Rows()
	if err != nil {
		t.Errorf("Not error should happen, got %v", err)
	}

	count := 0
	for rows.Next() {
		var name string
		var age int64
		rows.Scan(&name, &age)
		count++
	}

	if count != 2 {
		t.Errorf("Should found two records")
	}
}

func TestRaw(t *testing.T) {
	user1 := User{Name: "ExecRawSqlUser1", Age: 1}
	user2 := User{Name: "ExecRawSqlUser2", Age: 10}
	user3 := User{Name: "ExecRawSqlUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	type result struct {
		Name  string
		Email string
	}

	var results []result
	DB.Raw("SELECT name, age FROM users WHERE name = ? or name = ?", user2.Name, user3.Name).Scan(&results)
	if len(results) != 2 || results[0].Name != user2.Name || results[1].Name != user3.Name {
		t.Errorf("Raw with scan")
	}

	rows, _ := DB.Raw("select name, age from users where name = ?", user3.Name).Rows()
	count := 0
	for rows.Next() {
		count++
	}
	if count != 1 {
		t.Errorf("Raw with Rows should find one record with name 3")
	}

	DB.Exec("update users set name=? where name in (?)", "jinzhu", []string{user1.Name, user2.Name, user3.Name})
	if DB.Where("name in (?)", []string{user1.Name, user2.Name, user3.Name}).First(&User{}).Error != gorm.ErrRecordNotFound {
		t.Error("Raw sql to update records")
	}
}

func TestRowsWithGroup(t *testing.T) {
	users := []User{
		{Name: "having_user_1", Age: 1},
		{Name: "having_user_2", Age: 10},
		{Name: "having_user_1", Age: 20},
		{Name: "having_user_1", Age: 30},
	}

	DB.Create(&users)

	rows, err := DB.Select("name, count(*) as total").Table("users").Group("name").Having("name IN ?", []string{users[0].Name, users[1].Name}).Rows()
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		var total int64
		rows.Scan(&name, &total)

		if name == users[0].Name && total != 3 {
			t.Errorf("Should have one user having name %v", users[0].Name)
		} else if name == users[1].Name && total != 1 {
			t.Errorf("Should have two users having name %v", users[1].Name)
		}
	}
}

func TestQueryRaw(t *testing.T) {
	users := []*User{
		GetUser("row_query_user", Config{}),
		GetUser("row_query_user", Config{}),
		GetUser("row_query_user", Config{}),
	}
	DB.Create(&users)

	var user User
	DB.Raw("select * from users WHERE id = ?", users[1].ID).First(&user)
	CheckUser(t, user, *users[1])
}
