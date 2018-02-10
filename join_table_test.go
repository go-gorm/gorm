package gorm_test

import (
	"fmt"
	"strconv"
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
	DeletedAt *time.Time
	CreatedAt time.Time
}

func (*PersonAddress) Add(handler gorm.JoinTableHandlerInterface, db *gorm.DB, foreignValue interface{}, associationValue interface{}) error {
	foreignPrimaryKey, _ := strconv.Atoi(fmt.Sprint(db.NewScope(foreignValue).PrimaryKeyValue()))
	associationPrimaryKey, _ := strconv.Atoi(fmt.Sprint(db.NewScope(associationValue).PrimaryKeyValue()))
	if result := db.Unscoped().Model(&PersonAddress{}).Where(map[string]interface{}{
		"person_id":  foreignPrimaryKey,
		"address_id": associationPrimaryKey,
	}).Update(map[string]interface{}{
		"person_id":  foreignPrimaryKey,
		"address_id": associationPrimaryKey,
		"deleted_at": gorm.Expr("NULL"),
	}).RowsAffected; result == 0 {
		return db.Create(&PersonAddress{
			PersonID:  foreignPrimaryKey,
			AddressID: associationPrimaryKey,
		}).Error
	}

	return nil
}

func (*PersonAddress) Delete(handler gorm.JoinTableHandlerInterface, db *gorm.DB, sources ...interface{}) error {
	return db.Delete(&PersonAddress{}).Error
}

func (pa *PersonAddress) JoinWith(handler gorm.JoinTableHandlerInterface, db *gorm.DB, source interface{}) *gorm.DB {
	table := pa.Table(db)
	return db.Joins("INNER JOIN person_addresses ON person_addresses.address_id = addresses.id").Where(fmt.Sprintf("%v.deleted_at IS NULL OR %v.deleted_at <= '0001-01-02'", table, table))
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

func TestEmbeddedMany2ManyRelationship(t *testing.T) {
	type EmbeddedPerson struct {
		ID        int
		Name      string
		Addresses []*Address `gorm:"many2many:person_addresses;"`
	}

	type NewPerson struct {
		EmbeddedPerson
		ExternalID uint
	}
	DB.Exec("drop table person_addresses;")
	DB.AutoMigrate(&NewPerson{})

	address1 := &Address{Address1: "address 1"}
	address2 := &Address{Address1: "address 2"}
	person := &NewPerson{ExternalID: 100, EmbeddedPerson: EmbeddedPerson{Name: "person", Addresses: []*Address{address1, address2}}}
	if err := DB.Save(person).Error; err != nil {
		t.Errorf("no error should return when save embedded many2many relationship, but got %v", err)
	}

	if err := DB.Model(person).Association("Addresses").Delete(address1).Error; err != nil {
		t.Errorf("no error should return when delete embedded many2many relationship, but got %v", err)
	}

	association := DB.Model(person).Association("Addresses")
	if count := association.Count(); count != 1 || association.Error != nil {
		t.Errorf("Should found one address, but got %v, error is %v", count, association.Error)
	}

	if association.Clear(); association.Count() != 0 {
		t.Errorf("Should deleted all addresses")
	}
}
