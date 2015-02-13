package gorm_test

import "testing"

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

func TestPolymorphic(t *testing.T) {
	DB.AutoMigrate(&Cat{})
	DB.AutoMigrate(&Dog{})
	DB.AutoMigrate(&Toy{})

	cat := Cat{Name: "Mr. Bigglesworth", Toy: Toy{Name: "cat nip"}}
	dog := Dog{Name: "Pluto", Toys: []Toy{Toy{Name: "orange ball"}, Toy{Name: "yellow ball"}}}
	DB.Save(&cat).Save(&dog)

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

	if DB.Model(&cat).Association("Toy").Count() != 1 {
		t.Errorf("Should return one polymorphic has one association")
	}

	if DB.Model(&dog).Association("Toys").Count() != 2 {
		t.Errorf("Should return two polymorphic has many associations")
	}
}
