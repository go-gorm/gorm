package tests_test

import (
	. "gorm.io/gorm/utils/tests"
	"testing"
)

func TestRawSelect(t *testing.T) {
	users := []User{
		*GetUser("raw1", Config{}),
		*GetUser("raw2", Config{}),
		*GetUser("raw3", Config{}),
		*GetUser("@name", Config{}),
		*GetUser("@age", Config{}),
	}

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create users: %v", err)
	}
	tests := []struct {
		TestName string
		Sql      string
		args     map[string]interface{}
		Expect   []User
	}{
		{
			"raw_test1",
			`select * from users where name like @name and age = 18`,
			map[string]interface{}{
				"name": "raw1",
			},
			[]User{
				users[0],
			},
		},
		{
			"raw_test2",
			`select * from users where name like @name and age = 18`,
			map[string]interface{}{
				"name": "@name",
			},
			[]User{
				users[3],
			},
		},
		{
			"raw_test3",
			`select * from users where name like @name and age = 18`,
			map[string]interface{}{
				"name": "@age",
			},
			[]User{
				users[4],
			},
		},
		{
			"raw_test4",
			`select * from users where name like "@name" and age = 18`,
			map[string]interface{}{
				"name": "raw1",
			},
			[]User{
				users[3],
			},
		},
		{
			"raw_test5",
			`select * from users where name like "@name" and age = 18`,
			map[string]interface{}{
				"name": "@raw",
			},
			[]User{
				users[3],
			},
		},
		{
			"raw_test6",
			`select * from users where name like "@name" and age = 18`,
			map[string]interface{}{
				"name": "@age",
			},
			[]User{
				users[3],
			},
		},
	}
	for _, test := range tests {
		t.Run(test.TestName, func(t *testing.T) {
			var results []User
			if err := DB.Raw(test.Sql, test.args).Scan(&results).Error; err != nil {
				t.Errorf("errors %s: %v", test.TestName, err)
			} else {
				if len(results) != len(test.Expect) {
					t.Errorf("errors %s: %v", test.TestName, err)
				} else {
					for i := 0; i < len(results); i++ {
						CheckUser(t, results[i], test.Expect[i])
					}
				}
			}
		})
	}
}
