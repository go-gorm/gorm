package tests_test

import (
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func TestScan(t *testing.T) {
	user1 := User{Name: "ScanUser1", Age: 1}
	user2 := User{Name: "ScanUser2", Age: 10}
	user3 := User{Name: "ScanUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	type result struct {
		Name string
		Age  int
	}

	var res result
	DB.Table("users").Select("name, age").Where("id = ?", user3.ID).Scan(&res)
	if res.Name != user3.Name || res.Age != int(user3.Age) {
		t.Errorf("Scan into struct should work")
	}

	var doubleAgeRes = &result{}
	if err := DB.Debug().Table("users").Select("age + age as age").Where("id = ?", user3.ID).Scan(&doubleAgeRes).Error; err != nil {
		t.Errorf("Scan to pointer of pointer")
	}

	if doubleAgeRes.Age != int(res.Age)*2 {
		t.Errorf("Scan double age as age, expect: %v, got %v", res.Age*2, doubleAgeRes.Age)
	}

	var ress []result
	DB.Table("users").Select("name, age").Where("id in ?", []uint{user2.ID, user3.ID}).Scan(&ress)
	if len(ress) != 2 || ress[0].Name != user2.Name || ress[1].Name != user3.Name {
		t.Errorf("Scan into struct map")
	}
}
