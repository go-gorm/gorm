package tests_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	. "gorm.io/gorm/tests"
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
	if err := DB.Table("users").Select("age + age as age").Where("id = ?", user3.ID).Scan(&doubleAgeRes).Error; err != nil {
		t.Errorf("Scan to pointer of pointer")
	}

	if doubleAgeRes.Age != int(res.Age)*2 {
		t.Errorf("Scan double age as age, expect: %v, got %v", res.Age*2, doubleAgeRes.Age)
	}

	var results []result
	DB.Table("users").Select("name, age").Where("id in ?", []uint{user2.ID, user3.ID}).Scan(&results)

	sort.Slice(results, func(i, j int) bool {
		return strings.Compare(results[i].Name, results[j].Name) < -1
	})

	if len(results) != 2 || results[0].Name != user2.Name || results[1].Name != user3.Name {
		t.Errorf("Scan into struct map")
	}
}

func TestScanRows(t *testing.T) {
	user1 := User{Name: "ScanRowsUser1", Age: 1}
	user2 := User{Name: "ScanRowsUser2", Age: 10}
	user3 := User{Name: "ScanRowsUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	rows, err := DB.Table("users").Where("name = ? or name = ?", user2.Name, user3.Name).Select("name, age").Rows()
	if err != nil {
		t.Errorf("Not error should happen, got %v", err)
	}

	type Result struct {
		Name string
		Age  int
	}

	var results []Result
	for rows.Next() {
		var result Result
		if err := DB.ScanRows(rows, &result); err != nil {
			t.Errorf("should get no error, but got %v", err)
		}
		results = append(results, result)
	}

	if !reflect.DeepEqual(results, []Result{{Name: "ScanRowsUser2", Age: 10}, {Name: "ScanRowsUser3", Age: 20}}) {
		t.Errorf("Should find expected results")
	}
}
