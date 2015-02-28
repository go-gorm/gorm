package gorm_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

type Person struct {
	Id        int
	Name      string
	Addresses []*Address `gorm:"many2many:person_addresses;"`
}

type PersonAddress struct {
	PersonID  int
	AddressID int
	DeletedAt time.Time
	CreatedAt time.Time
}

func (*PersonAddress) Add(db *gorm.DB, relationship *gorm.Relationship, foreignValue interface{}, associationValue interface{}) error {
	return db.Where(map[string]interface{}{
		relationship.ForeignDBName:            foreignValue,
		relationship.AssociationForeignDBName: associationValue,
	}).Assign(map[string]interface{}{
		relationship.ForeignFieldName:            foreignValue,
		relationship.AssociationForeignFieldName: associationValue,
		"DeletedAt":                              gorm.Expr("NULL"),
	}).FirstOrCreate(&PersonAddress{}).Error
}

func (*PersonAddress) Delete(db *gorm.DB, relationship *gorm.Relationship) error {
	return db.Delete(&PersonAddress{}).Error
}

func (*PersonAddress) Scope(db *gorm.DB, relationship *gorm.Relationship) *gorm.DB {
	return db.Where(fmt.Sprintf("%v.deleted_at IS NULL OR %v.deleted_at <= '0001-01-02'", relationship.JoinTable, relationship.JoinTable))
}

func TestJoinTable(t *testing.T) {
	DB.Exec("drop table person_addresses;")
	DB.AutoMigrate(&Person{})
	DB.SetJoinTableHandler(&PersonAddress{}, "person_addresses")

	address1 := &Address{Address1: "address 1"}
	address2 := &Address{Address1: "address 2"}
	person := &Person{Name: "person", Addresses: []*Address{address1, address2}}
	DB.Save(person)

	DB.Model(person).Association("Addresses").Delete(address1)

	if DB.Find(&[]PersonAddress{}, "person_id = ?", person.Id).RowsAffected != 1 {
		t.Errorf("Should found one address")
	}

	if DB.Model(person).Association("Addresses").Count() != 1 {
		t.Errorf("Should found one address")
	}

	if DB.Unscoped().Find(&[]PersonAddress{}, "person_id = ?", person.Id).RowsAffected != 2 {
		t.Errorf("Found two addresses with Unscoped")
	}

	if DB.Model(person).Association("Addresses").Clear(); DB.Model(person).Association("Addresses").Count() != 0 {
		t.Errorf("Should deleted all addresses")
	}
}
