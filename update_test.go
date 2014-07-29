package gorm_test

import (
	"testing"
	"time"
)

func TestUpdate(t *testing.T) {
	product1 := Product{Code: "123"}
	product2 := Product{Code: "234"}
	animal1 := Animal{Name: "Ferdinand"}
	animal2 := Animal{Name: "nerdz"}

	db.Save(&product1).Save(&product2).Update("code", "456")

	if product2.Code != "456" {
		t.Errorf("Record should be updated with update attributes")
	}

	db.Save(&animal1).Save(&animal2).Update("name", "Francis")

	if animal2.Name != "Francis" {
		t.Errorf("Record should be updated with update attributes")
	}

	db.First(&product1, product1.Id)
	db.First(&product2, product2.Id)
	updated_at1 := product1.UpdatedAt
	updated_at2 := product2.UpdatedAt

	db.First(&animal1, animal1.Counter)
	db.First(&animal2, animal2.Counter)
	animalUpdated_at1 := animal1.UpdatedAt
	animalUpdated_at2 := animal2.UpdatedAt

	var product3 Product
	db.First(&product3, product2.Id).Update("code", "456")
	if updated_at2.Format(time.RFC3339Nano) != product3.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should not be updated if nothing changed")
	}

	if db.First(&Product{}, "code = '123'").Error != nil {
		t.Errorf("Product 123 should not be updated")
	}

	if db.First(&Product{}, "code = '234'").Error == nil {
		t.Errorf("Product 234 should be changed to 456")
	}

	if db.First(&Product{}, "code = '456'").Error != nil {
		t.Errorf("Product 234 should be changed to 456")
	}

	var animal3 Animal
	db.First(&animal3, animal2.Counter).Update("Name", "Robert")

	if animalUpdated_at2.Format(time.RFC3339Nano) != animal2.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should not be updated if nothing changed")
	}

	if db.First(&Animal{}, "name = 'Ferdinand'").Error != nil {
		t.Errorf("Animal 'Ferdinand' should not be updated")
	}

	if db.First(&Animal{}, "name = 'nerdz'").Error == nil {
		t.Errorf("Animal 'nerdz' should be changed to 'Francis'")
	}

	if db.First(&Animal{}, "name = 'Robert'").Error != nil {
		t.Errorf("Animal 'nerdz' should be changed to 'Robert'")
	}

	db.Table("products").Where("code in (?)", []string{"123"}).Update("code", "789")

	var product4 Product
	db.First(&product4, product1.Id)
	if updated_at1.Format(time.RFC3339Nano) != product4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should be updated if something changed")
	}

	if db.First(&Product{}, "code = '123'").Error == nil {
		t.Errorf("Product 123 should be changed to 789")
	}

	if db.First(&Product{}, "code = '456'").Error != nil {
		t.Errorf("Product 456 should not be changed to 789")
	}

	if db.First(&Product{}, "code = '789'").Error != nil {
		t.Errorf("Product 123 should be changed to 789")
	}

	if db.Model(product2).Update("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update with CamelCase")
	}

	if db.Model(&product2).UpdateColumn("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update_column with CamelCase")
	}

	db.Table("animals").Where("name in (?)", []string{"Ferdinand"}).Update("name", "Franz")

	var animal4 Animal
	db.First(&animal4, animal1.Counter)
	if animalUpdated_at1.Format(time.RFC3339Nano) != animal4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("animalUpdated_at should be updated if something changed")
	}

	if db.First(&Animal{}, "name = 'Ferdinand'").Error == nil {
		t.Errorf("Animal 'Ferdinand' should be changed to 'Franz'")
	}

	if db.First(&Animal{}, "name = 'Robert'").Error != nil {
		t.Errorf("Animal 'Robert' should not be changed to 'Francis'")
	}

	if db.First(&Animal{}, "name = 'Franz'").Error != nil {
		t.Errorf("Animal 'nerdz' should be changed to 'Franz'")
	}

	if db.Model(animal2).Update("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update with CamelCase")
	}

	if db.Model(&animal2).UpdateColumn("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update_column with CamelCase")
	}

	var animals []Animal
	db.Find(&animals)
	if count := db.Model(Animal{}).Update("CreatedAt", time.Now().Add(2*time.Hour)).RowsAffected; count != int64(len(animals)) {
		t.Error("RowsAffected should be correct when do batch update")
	}
}

func TestUpdates(t *testing.T) {
	product1 := Product{Code: "abc", Price: 10}
	product2 := Product{Code: "cde", Price: 20}
	db.Save(&product1).Save(&product2)
	db.Model(&product2).Updates(map[string]interface{}{"code": "edf", "price": 100})
	if product2.Code != "edf" || product2.Price != 100 {
		t.Errorf("Record should be updated also with update attributes")
	}

	db.First(&product1, product1.Id)
	db.First(&product2, product2.Id)
	updated_at1 := product1.UpdatedAt
	updated_at2 := product2.UpdatedAt

	var product3 Product
	db.First(&product3, product2.Id).Updates(Product{Code: "edf", Price: 100})
	if product3.Code != "edf" || product3.Price != 100 {
		t.Errorf("Record should be updated with update attributes")
	}

	if updated_at2.Format(time.RFC3339Nano) != product3.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should not be updated if nothing changed")
	}

	if db.First(&Product{}, "code = 'abc' and price = 10").Error != nil {
		t.Errorf("Product abc should not be updated")
	}

	if db.First(&Product{}, "code = 'cde'").Error == nil {
		t.Errorf("Product cde should be renamed to edf")
	}

	if db.First(&Product{}, "code = 'edf' and price = 100").Error != nil {
		t.Errorf("Product cde should be renamed to edf")
	}

	db.Table("products").Where("code in (?)", []string{"abc"}).Updates(map[string]string{"code": "fgh", "price": "200"})
	if db.First(&Product{}, "code = 'abc'").Error == nil {
		t.Errorf("Product abc's code should be changed to fgh")
	}

	var product4 Product
	db.First(&product4, product1.Id)
	if updated_at1.Format(time.RFC3339Nano) != product4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should be updated if something changed")
	}

	if db.First(&Product{}, "code = 'edf' and price = ?", 100).Error != nil {
		t.Errorf("Product cde's code should not be changed to fgh")
	}

	if db.First(&Product{}, "code = 'fgh' and price = 200").Error != nil {
		t.Errorf("We should have Product fgh")
	}
}

func TestUpdateColumn(t *testing.T) {
	product1 := Product{Code: "update_column 1", Price: 10}
	product2 := Product{Code: "update_column 2", Price: 20}
	db.Save(&product1).Save(&product2).UpdateColumn(map[string]interface{}{"code": "update_column 3", "price": 100})
	if product2.Code != "update_column 3" || product2.Price != 100 {
		t.Errorf("product 2 should be updated with update column")
	}

	var product3 Product
	db.First(&product3, product1.Id)
	if product3.Code != "update_column 1" || product3.Price != 10 {
		t.Errorf("product 1 should not be updated")
	}

	var product4, product5 Product
	db.First(&product4, product2.Id)
	updated_at1 := product4.UpdatedAt

	db.Model(Product{}).Where(product2.Id).UpdateColumn("code", "update_column_new")
	db.First(&product5, product2.Id)
	if updated_at1.Format(time.RFC3339Nano) != product5.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updated_at should not be updated with update column")
	}
}
