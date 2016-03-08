package gorm_test

import (
	"reflect"
	"sort"
	"testing"
)

type Cat struct {
	Id   int
	Name string
	Toy  Toy `gorm:"polymorphic:Owner;"`
}

type Dog struct {
	Id   int
	Name string
	Toys []Toy `gorm:"polymorphic:Owner;"`
}

type Toy struct {
	Id        int
	Name      string
	OwnerId   int
	OwnerType string
}

var compareToys = func(toys []Toy, contents []string) bool {
	var toyContents []string
	for _, toy := range toys {
		toyContents = append(toyContents, toy.Name)
	}
	sort.Strings(toyContents)
	sort.Strings(contents)
	return reflect.DeepEqual(toyContents, contents)
}

func TestPolymorphic(t *testing.T) {
	cat := Cat{Name: "Mr. Bigglesworth", Toy: Toy{Name: "cat toy"}}
	dog := Dog{Name: "Pluto", Toys: []Toy{{Name: "dog toy 1"}, {Name: "dog toy 2"}}}
	DB.Save(&cat).Save(&dog)

	if DB.Model(&cat).Association("Toy").Count() != 1 {
		t.Errorf("Cat's toys count should be 1")
	}

	if DB.Model(&dog).Association("Toys").Count() != 2 {
		t.Errorf("Dog's toys count should be 2")
	}

	// Query
	var catToys []Toy
	if DB.Model(&cat).Related(&catToys, "Toy").RecordNotFound() {
		t.Errorf("Did not find any has one polymorphic association")
	} else if len(catToys) != 1 {
		t.Errorf("Should have found only one polymorphic has one association")
	} else if catToys[0].Name != cat.Toy.Name {
		t.Errorf("Should have found the proper has one polymorphic association")
	}

	var dogToys []Toy
	if DB.Model(&dog).Related(&dogToys, "Toys").RecordNotFound() {
		t.Errorf("Did not find any polymorphic has many associations")
	} else if len(dogToys) != len(dog.Toys) {
		t.Errorf("Should have found all polymorphic has many associations")
	}

	var catToy Toy
	DB.Model(&cat).Association("Toy").Find(&catToy)
	if catToy.Name != cat.Toy.Name {
		t.Errorf("Should find has one polymorphic association")
	}

	var dogToys1 []Toy
	DB.Model(&dog).Association("Toys").Find(&dogToys1)
	if !compareToys(dogToys1, []string{"dog toy 1", "dog toy 2"}) {
		t.Errorf("Should find has many polymorphic association")
	}

	// Append
	DB.Model(&cat).Association("Toy").Append(&Toy{
		Name: "cat toy 2",
	})

	var catToy2 Toy
	DB.Model(&cat).Association("Toy").Find(&catToy2)
	if catToy2.Name != "cat toy 2" {
		t.Errorf("Should update has one polymorphic association with Append")
	}

	if DB.Model(&cat).Association("Toy").Count() != 1 {
		t.Errorf("Cat's toys count should be 1 after Append")
	}

	if DB.Model(&dog).Association("Toys").Count() != 2 {
		t.Errorf("Should return two polymorphic has many associations")
	}

	DB.Model(&dog).Association("Toys").Append(&Toy{
		Name: "dog toy 3",
	})

	var dogToys2 []Toy
	DB.Model(&dog).Association("Toys").Find(&dogToys2)
	if !compareToys(dogToys2, []string{"dog toy 1", "dog toy 2", "dog toy 3"}) {
		t.Errorf("Dog's toys should be updated with Append")
	}

	if DB.Model(&dog).Association("Toys").Count() != 3 {
		t.Errorf("Should return three polymorphic has many associations")
	}

	// Replace
	DB.Model(&cat).Association("Toy").Replace(&Toy{
		Name: "cat toy 3",
	})

	var catToy3 Toy
	DB.Model(&cat).Association("Toy").Find(&catToy3)
	if catToy3.Name != "cat toy 3" {
		t.Errorf("Should update has one polymorphic association with Replace")
	}

	if DB.Model(&cat).Association("Toy").Count() != 1 {
		t.Errorf("Cat's toys count should be 1 after Replace")
	}

	if DB.Model(&dog).Association("Toys").Count() != 3 {
		t.Errorf("Should return three polymorphic has many associations")
	}

	DB.Model(&dog).Association("Toys").Replace(&Toy{
		Name: "dog toy 4",
	}, []Toy{
		{Name: "dog toy 5"}, {Name: "dog toy 6"}, {Name: "dog toy 7"},
	})

	var dogToys3 []Toy
	DB.Model(&dog).Association("Toys").Find(&dogToys3)
	if !compareToys(dogToys3, []string{"dog toy 4", "dog toy 5", "dog toy 6", "dog toy 7"}) {
		t.Errorf("Dog's toys should be updated with Replace")
	}

	if DB.Model(&dog).Association("Toys").Count() != 4 {
		t.Errorf("Should return three polymorphic has many associations")
	}

	// Delete
	DB.Model(&cat).Association("Toy").Delete(&catToy2)

	var catToy4 Toy
	DB.Model(&cat).Association("Toy").Find(&catToy4)
	if catToy4.Name != "cat toy 3" {
		t.Errorf("Should not update has one polymorphic association when Delete a unrelated Toy")
	}

	if DB.Model(&cat).Association("Toy").Count() != 1 {
		t.Errorf("Cat's toys count should be 1")
	}

	if DB.Model(&dog).Association("Toys").Count() != 4 {
		t.Errorf("Dog's toys count should be 4")
	}

	DB.Model(&cat).Association("Toy").Delete(&catToy3)

	if !DB.Model(&cat).Related(&Toy{}, "Toy").RecordNotFound() {
		t.Errorf("Toy should be deleted with Delete")
	}

	if DB.Model(&cat).Association("Toy").Count() != 0 {
		t.Errorf("Cat's toys count should be 0 after Delete")
	}

	if DB.Model(&dog).Association("Toys").Count() != 4 {
		t.Errorf("Dog's toys count should not be changed when delete cat's toy")
	}

	DB.Model(&dog).Association("Toys").Delete(&dogToys2)

	if DB.Model(&dog).Association("Toys").Count() != 4 {
		t.Errorf("Dog's toys count should not be changed when delete unrelated toys")
	}

	DB.Model(&dog).Association("Toys").Delete(&dogToys3)

	if DB.Model(&dog).Association("Toys").Count() != 0 {
		t.Errorf("Dog's toys count should be deleted with Delete")
	}

	// Clear
	DB.Model(&cat).Association("Toy").Append(&Toy{
		Name: "cat toy 2",
	})

	if DB.Model(&cat).Association("Toy").Count() != 1 {
		t.Errorf("Cat's toys should be added with Append")
	}

	DB.Model(&cat).Association("Toy").Clear()

	if DB.Model(&cat).Association("Toy").Count() != 0 {
		t.Errorf("Cat's toys should be cleared with Clear")
	}

	DB.Model(&dog).Association("Toys").Append(&Toy{
		Name: "dog toy 8",
	})

	if DB.Model(&dog).Association("Toys").Count() != 1 {
		t.Errorf("Dog's toys should be added with Append")
	}

	DB.Model(&dog).Association("Toys").Clear()

	if DB.Model(&dog).Association("Toys").Count() != 0 {
		t.Errorf("Dog's toys should be cleared with Clear")
	}
}
