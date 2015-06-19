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
	gorm.JoinTableHandler
	PersonID  int
	AddressID int
	DeletedAt time.Time
	CreatedAt time.Time
}

func (*PersonAddress) Add(handler gorm.JoinTableHandlerInterface, db *gorm.DB, foreignValue interface{}, associationValue interface{}) error {
	return db.Where(map[string]interface{}{
		"person_id":  db.NewScope(foreignValue).PrimaryKeyValue(),
		"address_id": db.NewScope(associationValue).PrimaryKeyValue(),
	}).Assign(map[string]interface{}{
		"person_id":  foreignValue,
		"address_id": associationValue,
		"deleted_at": gorm.Expr("NULL"),
	}).FirstOrCreate(&PersonAddress{}).Error
}

func (*PersonAddress) Delete(handler gorm.JoinTableHandlerInterface, db *gorm.DB, sources ...interface{}) error {
	return db.Delete(&PersonAddress{}).Error
}

func (pa *PersonAddress) JoinWith(handler gorm.JoinTableHandlerInterface, db *gorm.DB, source interface{}) *gorm.DB {
	table := pa.Table(db)
	return db.Table(table).Joins("INNER JOIN person_addresses ON person_addresses.address_id = addresses.id").Where(fmt.Sprintf("%v.deleted_at IS NULL OR %v.deleted_at <= '0001-01-02'", table, table))
}

func TestJoinTable(t *testing.T) {
	DB.Exec("drop table person_addresses;")
	DB.AutoMigrate(&Person{})
	DB.SetJoinTableHandler(&Person{}, "Addresses", &PersonAddress{})

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
