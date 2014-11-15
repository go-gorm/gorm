package gorm_test

import (
	"testing"
	"time"
)

func TestUpdate(t *testing.T) {
	product1 := Product{Code: "product1code"}
	product2 := Product{Code: "product2code"}

	DB.Save(&product1).Save(&product2).Update("code", "product2newcode")

	if product2.Code != "product2newcode" {
		t.Errorf("Record should be updated")
	}

	DB.First(&product1, product1.Id)
	DB.First(&product2, product2.Id)
	updatedAt1 := product1.UpdatedAt
	updatedAt2 := product2.UpdatedAt

	var product3 Product
	DB.First(&product3, product2.Id).Update("code", "product2newcode")
	if updatedAt2.Format(time.RFC3339Nano) != product3.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updatedAt should not be updated if nothing changed")
	}

	if DB.First(&Product{}, "code = ?", product1.Code).RecordNotFound() {
		t.Errorf("Product1 should not be updated")
	}

	if !DB.First(&Product{}, "code = ?", "product2code").RecordNotFound() {
		t.Errorf("Product2's code should be updated")
	}

	if DB.First(&Product{}, "code = ?", "product2newcode").RecordNotFound() {
		t.Errorf("Product2's code should be updated")
	}

	DB.Table("products").Where("code in (?)", []string{"product1code"}).Update("code", "product1newcode")

	var product4 Product
	DB.First(&product4, product1.Id)
	if updatedAt1.Format(time.RFC3339Nano) != product4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updatedAt should be updated if something changed")
	}

	if !DB.First(&Product{}, "code = 'product1code'").RecordNotFound() {
		t.Errorf("Product1's code should be updated")
	}

	if DB.First(&Product{}, "code = 'product1newcode'").RecordNotFound() {
		t.Errorf("Product should not be changed to 789")
	}

	if DB.Model(product2).Update("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update with CamelCase")
	}

	if DB.Model(&product2).UpdateColumn("CreatedAt", time.Now().Add(time.Hour)).Error != nil {
		t.Error("No error should raise when update_column with CamelCase")
	}

	var products []Product
	DB.Find(&products)
	if count := DB.Model(Product{}).Update("CreatedAt", time.Now().Add(2*time.Hour)).RowsAffected; count != int64(len(products)) {
		t.Error("RowsAffected should be correct when do batch update")
	}
}

func TestUpdateWithNoStdPrimaryKeyAndDefaultValues(t *testing.T) {
	animal := Animal{Name: "Ferdinand"}
	DB.Save(&animal)
	updatedAt1 := animal.UpdatedAt

	DB.Save(&animal).Update("name", "Francis")

	if updatedAt1.Format(time.RFC3339Nano) == animal.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updatedAt should not be updated if nothing changed")
	}

	var animals []Animal
	DB.Find(&animals)
	if count := DB.Model(Animal{}).Update("CreatedAt", time.Now().Add(2*time.Hour)).RowsAffected; count != int64(len(animals)) {
		t.Error("RowsAffected should be correct when do batch update")
	}

	animal = Animal{From: "somewhere"}              // No name fields, should be filled with the default value (galeone)
	DB.Save(&animal).Update("From", "a nice place") // The name field shoul be untouched
	DB.First(&animal, animal.Counter)
	if animal.Name != "galeone" {
		t.Errorf("Name fiels shouldn't be changed if untouched, but got %v", animal.Name)
	}

	// When changing a field with a default value, the change must occur
	animal.Name = "amazing horse"
	DB.Save(&animal)
	DB.First(&animal, animal.Counter)
	if animal.Name != "amazing horse" {
		t.Errorf("Update a filed with a default value should occur. But got %v\n", animal.Name)
	}
}

func TestUpdates(t *testing.T) {
	product1 := Product{Code: "product1code", Price: 10}
	product2 := Product{Code: "product2code", Price: 10}
	DB.Save(&product1).Save(&product2)
	DB.Model(&product1).Updates(map[string]interface{}{"code": "product1newcode", "price": 100})
	if product1.Code != "product1newcode" || product1.Price != 100 {
		t.Errorf("Record should be updated also with map")
	}

	DB.First(&product1, product1.Id)
	DB.First(&product2, product2.Id)
	updatedAt1 := product1.UpdatedAt
	updatedAt2 := product2.UpdatedAt

	var product3 Product
	DB.First(&product3, product1.Id).Updates(Product{Code: "product1newcode", Price: 100})
	if product3.Code != "product1newcode" || product3.Price != 100 {
		t.Errorf("Record should be updated with struct")
	}

	if updatedAt1.Format(time.RFC3339Nano) != product3.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updatedAt should not be updated if nothing changed")
	}

	if DB.First(&Product{}, "code = ? and price = ?", product2.Code, product2.Price).RecordNotFound() {
		t.Errorf("Product2 should not be updated")
	}

	if DB.First(&Product{}, "code = ?", "product1newcode").RecordNotFound() {
		t.Errorf("Product1 should be updated")
	}

	DB.Table("products").Where("code in (?)", []string{"product2code"}).Updates(Product{Code: "product2newcode"})
	if !DB.First(&Product{}, "code = 'product2code'").RecordNotFound() {
		t.Errorf("Product2's code should be updated")
	}

	var product4 Product
	DB.First(&product4, product2.Id)
	if updatedAt2.Format(time.RFC3339Nano) != product4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updatedAt should be updated if something changed")
	}

	if DB.First(&Product{}, "code = ?", "product2newcode").RecordNotFound() {
		t.Errorf("product2's code should be updated")
	}
}

func TestUpdateColumn(t *testing.T) {
	product1 := Product{Code: "product1code", Price: 10}
	product2 := Product{Code: "product2code", Price: 20}
	DB.Save(&product1).Save(&product2).UpdateColumn(map[string]interface{}{"code": "product2newcode", "price": 100})
	if product2.Code != "product2newcode" || product2.Price != 100 {
		t.Errorf("product 2 should be updated with update column")
	}

	var product3 Product
	DB.First(&product3, product1.Id)
	if product3.Code != "product1code" || product3.Price != 10 {
		t.Errorf("product 1 should not be updated")
	}

	DB.First(&product2, product2.Id)
	updatedAt2 := product2.UpdatedAt
	DB.Model(product2).UpdateColumn("code", "update_column_new")
	var product4 Product
	DB.First(&product4, product2.Id)
	if updatedAt2.Format(time.RFC3339Nano) != product4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("updatedAt should not be updated with update column")
	}
}
