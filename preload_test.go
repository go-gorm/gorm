package gorm_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
)

func getPreloadUser(name string) *User {
	return getPreparedUser(name, "Preload")
}

func checkUserHasPreloadData(user User, t *testing.T) {
	u := getPreloadUser(user.Name)
	if user.BillingAddress.Address1 != u.BillingAddress.Address1 {
		t.Error("Failed to preload user's BillingAddress")
	}

	if user.ShippingAddress.Address1 != u.ShippingAddress.Address1 {
		t.Error("Failed to preload user's ShippingAddress")
	}

	if user.CreditCard.Number != u.CreditCard.Number {
		t.Error("Failed to preload user's CreditCard")
	}

	if user.Company.Name != u.Company.Name {
		t.Error("Failed to preload user's Company")
	}

	if len(user.Emails) != len(u.Emails) {
		t.Error("Failed to preload user's Emails")
	} else {
		var found int
		for _, e1 := range u.Emails {
			for _, e2 := range user.Emails {
				if e1.Email == e2.Email {
					found++
					break
				}
			}
		}
		if found != len(u.Emails) {
			t.Error("Failed to preload user's email details")
		}
	}
}

func TestPreload(t *testing.T) {
	user1 := getPreloadUser("user1")
	res1 := DB.Save(user1)
	if res1.Error != nil {
		t.Errorf("Error in save : %v", res1.Error)
	}

	preloadDB := DB.Where("role = ?", "Preload").Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company")
	if preloadDB.Error != nil {
		t.Errorf("Error in preload : %v", preloadDB.Error)
	}
	
	var user User
	res := preloadDB.Find(&user)
	if res.Error != nil {
		t.Errorf("Error in preload : %v", res.Error)
	}
	
	checkUserHasPreloadData(user, t)

	user2 := getPreloadUser("user2")
	DB.Save(user2)

	user3 := getPreloadUser("user3")
	DB.Save(user3)

	var users []User
	preloadDB.Find(&users)

	for _, user := range users {
		checkUserHasPreloadData(user, t)
	}

	var users2 []*User
	preloadDB.Find(&users2)

	for _, user := range users2 {
		checkUserHasPreloadData(*user, t)
	}

	var users3 []*User
	preloadDB.Preload("Emails", "email = ?", user3.Emails[0].Email).Find(&users3)

	for _, user := range users3 {
		if user.Name == user3.Name {
			if len(user.Emails) != 1 {
				t.Errorf("should only preload one emails for user3 when with condition")
			}
		} else if len(user.Emails) != 0 {
			t.Errorf("should not preload any emails for other users when with condition")
		}
	}
}

func TestNestedPreload1(t *testing.T) {
	type (
		Level1 struct {
			ID       uint
			Value    string
			Level2ID uint
		}
		Level2 struct {
			ID       uint
			Level1   Level1
			Level3ID uint
		}
		Level3 struct {
			ID     uint
			Name   string
			Level2 Level2
		}
	)
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := Level3{Level2: Level2{Level1: Level1{Value: "value"}}}
	if err := DB.Create(&want).Error; err != nil {
		panic(err)
	}

	var got Level3
	if err := DB.Preload("Level2").Preload("Level2.Level1").Find(&got).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}

	if err := DB.Preload("Level2").Preload("Level2.Level1").Find(&got, "name = ?", "not_found").Error; err != gorm.RecordNotFound {
		panic(err)
	}
}

func TestNestedPreload2(t *testing.T) {
	type (
		Level1 struct {
			ID       uint
			Value    string
			Level2ID uint
		}
		Level2 struct {
			ID       uint
			Level1s  []*Level1
			Level3ID uint
		}
		Level3 struct {
			ID      uint
			Name    string
			Level2s []Level2
		}
	)
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := Level3{
		Level2s: []Level2{
			{
				Level1s: []*Level1{
					&Level1{Value: "value1"},
					&Level1{Value: "value2"},
				},
			},
			{
				Level1s: []*Level1{
					&Level1{Value: "value3"},
				},
			},
		},
	}
	if err := DB.Create(&want).Error; err != nil {
		panic(err)
	}

	var got Level3
	if err := DB.Preload("Level2s.Level1s").Find(&got).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}
}

func TestNestedPreload3(t *testing.T) {
	type (
		Level1 struct {
			ID       uint
			Value    string
			Level2ID uint
		}
		Level2 struct {
			ID       uint
			Level1   Level1
			Level3ID uint
		}
		Level3 struct {
			Name    string
			ID      uint
			Level2s []Level2
		}
	)
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := Level3{
		Level2s: []Level2{
			{Level1: Level1{Value: "value1"}},
			{Level1: Level1{Value: "value2"}},
		},
	}
	if err := DB.Create(&want).Error; err != nil {
		panic(err)
	}

	var got Level3
	if err := DB.Preload("Level2s.Level1").Find(&got).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}
}

func TestNestedPreload4(t *testing.T) {
	type (
		Level1 struct {
			ID       uint
			Value    string
			Level2ID uint
		}
		Level2 struct {
			ID       uint
			Level1s  []Level1
			Level3ID uint
		}
		Level3 struct {
			ID     uint
			Name   string
			Level2 Level2
		}
	)
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := Level3{
		Level2: Level2{
			Level1s: []Level1{
				Level1{Value: "value1"},
				Level1{Value: "value2"},
			},
		},
	}
	if err := DB.Create(&want).Error; err != nil {
		panic(err)
	}

	var got Level3
	if err := DB.Preload("Level2.Level1s").Find(&got).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}
}

// Slice: []Level3
func TestNestedPreload5(t *testing.T) {
	type (
		Level1 struct {
			ID       uint
			Value    string
			Level2ID uint
		}
		Level2 struct {
			ID       uint
			Level1   Level1
			Level3ID uint
		}
		Level3 struct {
			ID     uint
			Name   string
			Level2 Level2
		}
	)
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := make([]Level3, 2)
	want[0] = Level3{Level2: Level2{Level1: Level1{Value: "value"}}}
	if err := DB.Create(&want[0]).Error; err != nil {
		panic(err)
	}
	want[1] = Level3{Level2: Level2{Level1: Level1{Value: "value2"}}}
	if err := DB.Create(&want[1]).Error; err != nil {
		panic(err)
	}

	var got []Level3
	if err := DB.Preload("Level2").Preload("Level2.Level1").Find(&got).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}
}

func TestNestedPreload6(t *testing.T) {
	type (
		Level1 struct {
			ID       uint
			Value    string
			Level2ID uint
		}
		Level2 struct {
			ID       uint
			Level1s  []Level1
			Level3ID uint
		}
		Level3 struct {
			ID      uint
			Name    string
			Level2s []Level2
		}
	)
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := make([]Level3, 2)
	want[0] = Level3{
		Level2s: []Level2{
			{
				Level1s: []Level1{
					{Value: "value1"},
					{Value: "value2"},
				},
			},
			{
				Level1s: []Level1{
					{Value: "value3"},
				},
			},
		},
	}
	if err := DB.Create(&want[0]).Error; err != nil {
		panic(err)
	}

	want[1] = Level3{
		Level2s: []Level2{
			{
				Level1s: []Level1{
					{Value: "value3"},
					{Value: "value4"},
				},
			},
			{
				Level1s: []Level1{
					{Value: "value5"},
				},
			},
		},
	}
	if err := DB.Create(&want[1]).Error; err != nil {
		panic(err)
	}

	var got []Level3
	if err := DB.Preload("Level2s.Level1s").Find(&got).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}
}

func TestNestedPreload7(t *testing.T) {
	type (
		Level1 struct {
			ID       uint
			Value    string
			Level2ID uint
		}
		Level2 struct {
			ID       uint
			Level1   Level1
			Level3ID uint
		}
		Level3 struct {
			ID      uint
			Name    string
			Level2s []Level2
		}
	)
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := make([]Level3, 2)
	want[0] = Level3{
		Level2s: []Level2{
			{Level1: Level1{Value: "value1"}},
			{Level1: Level1{Value: "value2"}},
		},
	}
	if err := DB.Create(&want[0]).Error; err != nil {
		panic(err)
	}

	want[1] = Level3{
		Level2s: []Level2{
			{Level1: Level1{Value: "value3"}},
			{Level1: Level1{Value: "value4"}},
		},
	}
	if err := DB.Create(&want[1]).Error; err != nil {
		panic(err)
	}

	var got []Level3
	if err := DB.Preload("Level2s.Level1").Find(&got).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}
}

func TestNestedPreload8(t *testing.T) {
	type (
		Level1 struct {
			ID       uint
			Value    string
			Level2ID uint
		}
		Level2 struct {
			ID       uint
			Level1s  []Level1
			Level3ID uint
		}
		Level3 struct {
			ID     uint
			Name   string
			Level2 Level2
		}
	)
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := make([]Level3, 2)
	want[0] = Level3{
		Level2: Level2{
			Level1s: []Level1{
				Level1{Value: "value1"},
				Level1{Value: "value2"},
			},
		},
	}
	if err := DB.Create(&want[0]).Error; err != nil {
		panic(err)
	}
	want[1] = Level3{
		Level2: Level2{
			Level1s: []Level1{
				Level1{Value: "value3"},
				Level1{Value: "value4"},
			},
		},
	}
	if err := DB.Create(&want[1]).Error; err != nil {
		panic(err)
	}

	var got []Level3
	if err := DB.Preload("Level2.Level1s").Find(&got).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}
}

func TestNestedPreload9(t *testing.T) {
	type (
		Level0 struct {
			ID       uint
			Value    string
			Level1ID uint
		}
		Level1 struct {
			ID         uint
			Value      string
			Level2ID   uint
			Level2_1ID uint
			Level0s    []Level0
		}
		Level2 struct {
			ID       uint
			Level1s  []Level1
			Level3ID uint
		}
		Level2_1 struct {
			ID       uint
			Level1s  []Level1
			Level3ID uint
		}
		Level3 struct {
			ID       uint
			Name     string
			Level2   Level2
			Level2_1 Level2_1
		}
	)
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level2_1{})
	DB.DropTableIfExists(&Level1{})
	DB.DropTableIfExists(&Level0{})
	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}, &Level2_1{}, &Level0{}).Error; err != nil {
		panic(err)
	}

	want := make([]Level3, 2)
	want[0] = Level3{
		Level2: Level2{
			Level1s: []Level1{
				Level1{Value: "value1"},
				Level1{Value: "value2"},
			},
		},
		Level2_1: Level2_1{
			Level1s: []Level1{
				Level1{
					Value:   "value1-1",
					Level0s: []Level0{{Value: "Level0-1"}},
				},
				Level1{
					Value:   "value2-2",
					Level0s: []Level0{{Value: "Level0-2"}},
				},
			},
		},
	}
	if err := DB.Create(&want[0]).Error; err != nil {
		panic(err)
	}
	want[1] = Level3{
		Level2: Level2{
			Level1s: []Level1{
				Level1{Value: "value3"},
				Level1{Value: "value4"},
			},
		},
		Level2_1: Level2_1{
			Level1s: []Level1{
				Level1{Value: "value3-3"},
				Level1{Value: "value4-4"},
			},
		},
	}
	if err := DB.Create(&want[1]).Error; err != nil {
		panic(err)
	}

	var got []Level3
	if err := DB.Preload("Level2").Preload("Level2.Level1s").Preload("Level2_1").Preload("Level2_1.Level1s").Preload("Level2_1.Level1s.Level0s").Find(&got).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}
}

func TestManyToManyPreloadWithMultiPrimaryKeys(t *testing.T) {
	if dialect := os.Getenv("GORM_DIALECT"); dialect == "" || dialect == "sqlite" {
		return
	}

	type (
		Level1 struct {
			ID           uint   `gorm:"primary_key;"`
			LanguageCode string `gorm:"primary_key"`
			Value        string
		}
		Level2 struct {
			ID           uint   `gorm:"primary_key;"`
			LanguageCode string `gorm:"primary_key"`
			Value        string
			Level1s      []Level1 `gorm:"many2many:levels;"`
		}
	)

	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	DB.DropTableIfExists("levels")

	if err := DB.AutoMigrate(&Level2{}, &Level1{}).Error; err != nil {
		t.Fatalf("Error in AutoMigrate:%v", err)
	}

	want := Level2{Value: "Bob", LanguageCode: "ru", Level1s: []Level1{
		{Value: "ru", LanguageCode: "ru"},
		{Value: "en", LanguageCode: "en"},
	}}
	if err := DB.Save(&want).Error; err != nil {
		t.Fatalf("Error in Save:%v", err)
	}

	want2 := Level2{Value: "Tom", LanguageCode: "zh", Level1s: []Level1{
		{Value: "zh", LanguageCode: "zh"},
		{Value: "de", LanguageCode: "de"},
	}}
	if err := DB.Save(&want2).Error; err != nil {
		t.Fatalf("Error in Save want2:%v", err)
	}

	var got Level2
	if err := DB.Preload("Level1s").Find(&got, "value = ?", "Bob").Error; err != nil {
		t.Fatalf("Error in Preload:%v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}

	var got2 Level2
	if err := DB.Preload("Level1s").Find(&got2, "value = ?", "Tom").Error; err != nil {
		t.Fatalf("Error in Preload Level1s:%v", err)
	}

	if !reflect.DeepEqual(got2, want2) {
		t.Errorf("got %s; want %s", toJSONString(got2), toJSONString(want2))
	}

	var got3 []Level2
	if err := DB.Preload("Level1s").Find(&got3, "value IN (?)", []string{"Bob", "Tom"}).Error; err != nil {
		t.Fatalf("Error in Preload got3 :%v",err)
	}

	if !reflect.DeepEqual(got3, []Level2{got, got2}) {
		t.Errorf("got %s; want %s", toJSONString(got3), toJSONString([]Level2{got, got2}))
	}

	var got4 []Level2
	if err := DB.Preload("Level1s", "value IN (?)", []string{"zh", "ru"}).Find(&got4, "value IN (?)", []string{"Bob", "Tom"}).Error; err != nil {
		t.Fatalf("Error in Preload got4:%v",err)
	}

	var ruLevel1 Level1
	var zhLevel1 Level1
	DB.First(&ruLevel1, "value = ?", "ru")
	DB.First(&zhLevel1, "value = ?", "zh")

	got.Level1s = []Level1{ruLevel1}
	got2.Level1s = []Level1{zhLevel1}
	if !reflect.DeepEqual(got4, []Level2{got, got2}) {
		t.Errorf("got %s; want %s", toJSONString(got4), toJSONString([]Level2{got, got2}))
	}

	if err := DB.Preload("Level1s").Find(&got4, "value IN (?)", []string{"non-existing"}).Error; err != nil {
		panic(err)
	}
}

func TestManyToManyPreloadForPointer(t *testing.T) {
	type (
		Level1 struct {
			ID    uint
			Value string
		}
		Level2 struct {
			ID      uint
			Value   string
			Level1s []*Level1 `gorm:"many2many:levels;"`
		}
	)

	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	DB.DropTableIfExists("levels")

	if err := DB.AutoMigrate(&Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := Level2{Value: "Bob", Level1s: []*Level1{
		{Value: "ru"},
		{Value: "en"},
	}}
	if err := DB.Save(&want).Error; err != nil {
		panic(err)
	}

	want2 := Level2{Value: "Tom", Level1s: []*Level1{
		{Value: "zh"},
		{Value: "de"},
	}}
	if err := DB.Save(&want2).Error; err != nil {
		panic(err)
	}

	var got Level2
	if err := DB.Preload("Level1s").Find(&got, "value = ?", "Bob").Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}

	var got2 Level2
	if err := DB.Preload("Level1s").Find(&got2, "value = ?", "Tom").Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got2, want2) {
		t.Errorf("got %s; want %s", toJSONString(got2), toJSONString(want2))
	}

	var got3 []Level2
	if err := DB.Preload("Level1s").Find(&got3, "value IN (?)", []string{"Bob", "Tom"}).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got3, []Level2{got, got2}) {
		t.Errorf("got %s; want %s", toJSONString(got3), toJSONString([]Level2{got, got2}))
	}

	var got4 []Level2
	if err := DB.Preload("Level1s", "value IN (?)", []string{"zh", "ru"}).Find(&got4, "value IN (?)", []string{"Bob", "Tom"}).Error; err != nil {
		panic(err)
	}

	var got5 Level2
	DB.Preload("Level1s").First(&got5, "value = ?", "bogus")

	var ruLevel1 Level1
	var zhLevel1 Level1
	DB.First(&ruLevel1, "value = ?", "ru")
	DB.First(&zhLevel1, "value = ?", "zh")

	got.Level1s = []*Level1{&ruLevel1}
	got2.Level1s = []*Level1{&zhLevel1}
	if !reflect.DeepEqual(got4, []Level2{got, got2}) {
		t.Errorf("got %s; want %s", toJSONString(got4), toJSONString([]Level2{got, got2}))
	}
}

func TestManyToManyPreloadForNestedPointer(t *testing.T) {
	type (
		Level1 struct {
			ID    uint
			Value string
		}
		Level2 struct {
			ID      uint
			Value   string
			Level1s []*Level1 `gorm:"many2many:levels;"`
		}
		Level3 struct {
			ID       uint
			Value    string
			Level2ID sql.NullInt64
			Level2   *Level2
		}
	)

	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})
	DB.DropTableIfExists("levels")

	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := Level3{
		Value: "Bob",
		Level2: &Level2{
			Value: "Foo",
			Level1s: []*Level1{
				{Value: "ru"},
				{Value: "en"},
			},
		},
	}
	if err := DB.Save(&want).Error; err != nil {
		panic(err)
	}

	want2 := Level3{
		Value: "Tom",
		Level2: &Level2{
			Value: "Bar",
			Level1s: []*Level1{
				{Value: "zh"},
				{Value: "de"},
			},
		},
	}
	if err := DB.Save(&want2).Error; err != nil {
		panic(err)
	}

	var got Level3
	if err := DB.Preload("Level2.Level1s").Find(&got, "value = ?", "Bob").Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}

	var got2 Level3
	if err := DB.Preload("Level2.Level1s").Find(&got2, "value = ?", "Tom").Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got2, want2) {
		t.Errorf("got %s; want %s", toJSONString(got2), toJSONString(want2))
	}

	var got3 []Level3
	if err := DB.Preload("Level2.Level1s").Find(&got3, "value IN (?)", []string{"Bob", "Tom"}).Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got3, []Level3{got, got2}) {
		t.Errorf("got %s; want %s", toJSONString(got3), toJSONString([]Level3{got, got2}))
	}

	var got4 []Level3
	if err := DB.Preload("Level2.Level1s", "value IN (?)", []string{"zh", "ru"}).Find(&got4, "value IN (?)", []string{"Bob", "Tom"}).Error; err != nil {
		panic(err)
	}

	var got5 Level3
	DB.Preload("Level2.Level1s").Find(&got5, "value = ?", "bogus")

	var ruLevel1 Level1
	var zhLevel1 Level1
	DB.First(&ruLevel1, "value = ?", "ru")
	DB.First(&zhLevel1, "value = ?", "zh")

	got.Level2.Level1s = []*Level1{&ruLevel1}
	got2.Level2.Level1s = []*Level1{&zhLevel1}
	if !reflect.DeepEqual(got4, []Level3{got, got2}) {
		t.Errorf("got %s; want %s", toJSONString(got4), toJSONString([]Level3{got, got2}))
	}
}

func TestNestedManyToManyPreload(t *testing.T) {
	type (
		Level1 struct {
			ID    uint
			Value string
		}
		Level2 struct {
			ID      uint
			Value   string
			Level1s []*Level1 `gorm:"many2many:level1_level2;"`
		}
		Level3 struct {
			ID      uint
			Value   string
			Level2s []Level2 `gorm:"many2many:level2_level3;"`
		}
	)

	DB.DropTableIfExists(&Level1{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists("level1_level2")
	DB.DropTableIfExists("level2_level3")

	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := Level3{
		Value: "Level3",
		Level2s: []Level2{
			{
				Value: "Bob",
				Level1s: []*Level1{
					{Value: "ru"},
					{Value: "en"},
				},
			}, {
				Value: "Tom",
				Level1s: []*Level1{
					{Value: "zh"},
					{Value: "de"},
				},
			},
		},
	}

	if err := DB.Save(&want).Error; err != nil {
		panic(err)
	}

	var got Level3
	if err := DB.Preload("Level2s").Preload("Level2s.Level1s").Find(&got, "value = ?", "Level3").Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}

	if err := DB.Preload("Level2s.Level1s").Find(&got, "value = ?", "not_found").Error; err != gorm.RecordNotFound {
		panic(err)
	}
}

func TestNestedManyToManyPreload2(t *testing.T) {
	type (
		Level1 struct {
			ID    uint
			Value string
		}
		Level2 struct {
			ID      uint
			Value   string
			Level1s []*Level1 `gorm:"many2many:level1_level2;"`
		}
		Level3 struct {
			ID       uint
			Value    string
			Level2ID sql.NullInt64
			Level2   *Level2
		}
	)

	DB.DropTableIfExists(&Level1{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists("level1_level2")

	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := Level3{
		Value: "Level3",
		Level2: &Level2{
			Value: "Bob",
			Level1s: []*Level1{
				{Value: "ru"},
				{Value: "en"},
			},
		},
	}

	if err := DB.Save(&want).Error; err != nil {
		panic(err)
	}

	var got Level3
	if err := DB.Preload("Level2.Level1s").Find(&got, "value = ?", "Level3").Error; err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s; want %s", toJSONString(got), toJSONString(want))
	}

	if err := DB.Preload("Level2.Level1s").Find(&got, "value = ?", "not_found").Error; err != gorm.RecordNotFound {
		panic(err)
	}
}

func TestNilPointerSlice(t *testing.T) {
	type (
		Level3 struct {
			ID    uint
			Value string
		}
		Level2 struct {
			ID       uint
			Value    string
			Level3ID uint
			Level3   *Level3
		}
		Level1 struct {
			ID       uint
			Value    string
			Level2ID uint
			Level2   *Level2
		}
	)

	DB.DropTableIfExists(&Level3{})
	DB.DropTableIfExists(&Level2{})
	DB.DropTableIfExists(&Level1{})

	if err := DB.AutoMigrate(&Level3{}, &Level2{}, &Level1{}).Error; err != nil {
		panic(err)
	}

	want := Level1{Value: "Bob", Level2: &Level2{
		Value: "en",
		Level3: &Level3{
			Value: "native",
		},
	}}
	if err := DB.Save(&want).Error; err != nil {
		panic(err)
	}

	want2 := Level1{Value: "Tom", Level2: nil}
	if err := DB.Save(&want2).Error; err != nil {
		panic(err)
	}

	var got []Level1
	if err := DB.Preload("Level2").Preload("Level2.Level3").Find(&got).Error; err != nil {
		panic(err)
	}

	if len(got) != 2 {
		t.Fatalf("got %v items, expected 2", len(got))
	}

	if !reflect.DeepEqual(got[0], want) && !reflect.DeepEqual(got[1], want) {
		t.Errorf("got %s; want array containing %s", toJSONString(got), toJSONString(want))
	}

	if !reflect.DeepEqual(got[0], want2) && !reflect.DeepEqual(got[1], want2) {
		t.Errorf("got %s; want array containing %s", toJSONString(got), toJSONString(want2))
	}
}

func toJSONString(v interface{}) []byte {
	r, _ := json.MarshalIndent(v, "", "  ")
	return r
}
