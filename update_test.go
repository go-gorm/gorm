package gorm_test

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
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

	DB.First(&product4, product4.Id)
	updatedAt4 := product4.UpdatedAt
	DB.Model(&product4).Update("price", gorm.Expr("price + ? - ?", 100, 50))
	var product5 Product
	DB.First(&product5, product4.Id)
	if product5.Price != product4.Price+100-50 {
		t.Errorf("Update with expression")
	}
	if product4.UpdatedAt.Format(time.RFC3339Nano) == updatedAt4.Format(time.RFC3339Nano) {
		t.Errorf("Update with expression should update UpdatedAt")
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
		t.Errorf("Name fields shouldn't be changed if untouched, but got %v", animal.Name)
	}

	// When changing a field with a default value, the change must occur
	animal.Name = "amazing horse"
	DB.Save(&animal)
	DB.First(&animal, animal.Counter)
	if animal.Name != "amazing horse" {
		t.Errorf("Update a filed with a default value should occur. But got %v\n", animal.Name)
	}

	// When changing a field with a default value with blank value
	animal.Name = ""
	DB.Save(&animal)
	DB.First(&animal, animal.Counter)
	if animal.Name != "" {
		t.Errorf("Update a filed to blank with a default value should occur. But got %v\n", animal.Name)
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
	updatedAt2 := product2.UpdatedAt

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

	updatedAt4 := product4.UpdatedAt
	DB.Model(&product4).Updates(map[string]interface{}{"price": gorm.Expr("price + ?", 100)})
	var product5 Product
	DB.First(&product5, product4.Id)
	if product5.Price != product4.Price+100 {
		t.Errorf("Updates with expression")
	}
	// product4's UpdatedAt will be reset when updating
	if product4.UpdatedAt.Format(time.RFC3339Nano) == updatedAt4.Format(time.RFC3339Nano) {
		t.Errorf("Updates with expression should update UpdatedAt")
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

	DB.Model(&product4).UpdateColumn("price", gorm.Expr("price + 100 - 50"))
	var product5 Product
	DB.First(&product5, product4.Id)
	if product5.Price != product4.Price+100-50 {
		t.Errorf("UpdateColumn with expression")
	}
	if product5.UpdatedAt.Format(time.RFC3339Nano) != product4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("UpdateColumn with expression should not update UpdatedAt")
	}
}

func TestSelectWithUpdate(t *testing.T) {
	user := getPreparedUser("select_user", "select_with_update")
	DB.Create(user)

	var reloadUser User
	DB.First(&reloadUser, user.Id)
	reloadUser.Name = "new_name"
	reloadUser.Age = 50
	reloadUser.BillingAddress = Address{Address1: "New Billing Address"}
	reloadUser.ShippingAddress = Address{Address1: "New ShippingAddress Address"}
	reloadUser.CreditCard = CreditCard{Number: "987654321"}
	reloadUser.Emails = []Email{
		{Email: "new_user_1@example1.com"}, {Email: "new_user_2@example2.com"}, {Email: "new_user_3@example2.com"},
	}
	reloadUser.Company = Company{Name: "new company"}

	DB.Select("Name", "BillingAddress", "CreditCard", "Company", "Emails").Save(&reloadUser)

	var queryUser User
	DB.Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company").First(&queryUser, user.Id)

	if queryUser.Name == user.Name || queryUser.Age != user.Age {
		t.Errorf("Should only update users with name column")
	}

	if queryUser.BillingAddressID.Int64 == user.BillingAddressID.Int64 ||
		queryUser.ShippingAddressId != user.ShippingAddressId ||
		queryUser.CreditCard.ID == user.CreditCard.ID ||
		len(queryUser.Emails) == len(user.Emails) || queryUser.Company.Id == user.Company.Id {
		t.Errorf("Should only update selected relationships")
	}
}

func TestSelectWithUpdateWithMap(t *testing.T) {
	user := getPreparedUser("select_user", "select_with_update_map")
	DB.Create(user)

	updateValues := map[string]interface{}{
		"Name":            "new_name",
		"Age":             50,
		"BillingAddress":  Address{Address1: "New Billing Address"},
		"ShippingAddress": Address{Address1: "New ShippingAddress Address"},
		"CreditCard":      CreditCard{Number: "987654321"},
		"Emails": []Email{
			{Email: "new_user_1@example1.com"}, {Email: "new_user_2@example2.com"}, {Email: "new_user_3@example2.com"},
		},
		"Company": Company{Name: "new company"},
	}

	var reloadUser User
	DB.First(&reloadUser, user.Id)
	DB.Model(&reloadUser).Select("Name", "BillingAddress", "CreditCard", "Company", "Emails").Update(updateValues)

	var queryUser User
	DB.Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company").First(&queryUser, user.Id)

	if queryUser.Name == user.Name || queryUser.Age != user.Age {
		t.Errorf("Should only update users with name column")
	}

	if queryUser.BillingAddressID.Int64 == user.BillingAddressID.Int64 ||
		queryUser.ShippingAddressId != user.ShippingAddressId ||
		queryUser.CreditCard.ID == user.CreditCard.ID ||
		len(queryUser.Emails) == len(user.Emails) || queryUser.Company.Id == user.Company.Id {
		t.Errorf("Should only update selected relationships")
	}
}

func TestOmitWithUpdate(t *testing.T) {
	user := getPreparedUser("omit_user", "omit_with_update")
	DB.Create(user)

	var reloadUser User
	DB.First(&reloadUser, user.Id)
	reloadUser.Name = "new_name"
	reloadUser.Age = 50
	reloadUser.BillingAddress = Address{Address1: "New Billing Address"}
	reloadUser.ShippingAddress = Address{Address1: "New ShippingAddress Address"}
	reloadUser.CreditCard = CreditCard{Number: "987654321"}
	reloadUser.Emails = []Email{
		{Email: "new_user_1@example1.com"}, {Email: "new_user_2@example2.com"}, {Email: "new_user_3@example2.com"},
	}
	reloadUser.Company = Company{Name: "new company"}

	DB.Omit("Name", "BillingAddress", "CreditCard", "Company", "Emails").Save(&reloadUser)

	var queryUser User
	DB.Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company").First(&queryUser, user.Id)

	if queryUser.Name != user.Name || queryUser.Age == user.Age {
		t.Errorf("Should only update users with name column")
	}

	if queryUser.BillingAddressID.Int64 != user.BillingAddressID.Int64 ||
		queryUser.ShippingAddressId == user.ShippingAddressId ||
		queryUser.CreditCard.ID != user.CreditCard.ID ||
		len(queryUser.Emails) != len(user.Emails) || queryUser.Company.Id != user.Company.Id {
		t.Errorf("Should only update relationships that not omitted")
	}
}

func TestOmitWithUpdateWithMap(t *testing.T) {
	user := getPreparedUser("select_user", "select_with_update_map")
	DB.Create(user)

	updateValues := map[string]interface{}{
		"Name":            "new_name",
		"Age":             50,
		"BillingAddress":  Address{Address1: "New Billing Address"},
		"ShippingAddress": Address{Address1: "New ShippingAddress Address"},
		"CreditCard":      CreditCard{Number: "987654321"},
		"Emails": []Email{
			{Email: "new_user_1@example1.com"}, {Email: "new_user_2@example2.com"}, {Email: "new_user_3@example2.com"},
		},
		"Company": Company{Name: "new company"},
	}

	var reloadUser User
	DB.First(&reloadUser, user.Id)
	DB.Model(&reloadUser).Omit("Name", "BillingAddress", "CreditCard", "Company", "Emails").Update(updateValues)

	var queryUser User
	DB.Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company").First(&queryUser, user.Id)

	if queryUser.Name != user.Name || queryUser.Age == user.Age {
		t.Errorf("Should only update users with name column")
	}

	if queryUser.BillingAddressID.Int64 != user.BillingAddressID.Int64 ||
		queryUser.ShippingAddressId == user.ShippingAddressId ||
		queryUser.CreditCard.ID != user.CreditCard.ID ||
		len(queryUser.Emails) != len(user.Emails) || queryUser.Company.Id != user.Company.Id {
		t.Errorf("Should only update relationships not omitted")
	}
}

func TestSelectWithUpdateColumn(t *testing.T) {
	user := getPreparedUser("select_user", "select_with_update_map")
	DB.Create(user)

	updateValues := map[string]interface{}{"Name": "new_name", "Age": 50}

	var reloadUser User
	DB.First(&reloadUser, user.Id)
	DB.Model(&reloadUser).Select("Name").UpdateColumn(updateValues)

	var queryUser User
	DB.First(&queryUser, user.Id)

	if queryUser.Name == user.Name || queryUser.Age != user.Age {
		t.Errorf("Should only update users with name column")
	}
}

func TestOmitWithUpdateColumn(t *testing.T) {
	user := getPreparedUser("select_user", "select_with_update_map")
	DB.Create(user)

	updateValues := map[string]interface{}{"Name": "new_name", "Age": 50}

	var reloadUser User
	DB.First(&reloadUser, user.Id)
	DB.Model(&reloadUser).Omit("Name").UpdateColumn(updateValues)

	var queryUser User
	DB.First(&queryUser, user.Id)

	if queryUser.Name != user.Name || queryUser.Age == user.Age {
		t.Errorf("Should omit name column when update user")
	}
}

func TestUpdateColumnsSkipsAssociations(t *testing.T) {
	user := getPreparedUser("update_columns_user", "special_role")
	user.Age = 99
	address1 := "first street"
	user.BillingAddress = Address{Address1: address1}
	DB.Save(user)

	// Update a single field of the user and verify that the changed address is not stored.
	newAge := int64(100)
	user.BillingAddress.Address1 = "second street"
	db := DB.Model(user).UpdateColumns(User{Age: newAge})
	if db.RowsAffected != 1 {
		t.Errorf("Expected RowsAffected=1 but instead RowsAffected=%v", DB.RowsAffected)
	}

	// Verify that Age now=`newAge`.
	freshUser := &User{Id: user.Id}
	DB.First(freshUser)
	if freshUser.Age != newAge {
		t.Errorf("Expected freshly queried user to have Age=%v but instead found Age=%v", newAge, freshUser.Age)
	}

	// Verify that user's BillingAddress.Address1 is not changed and is still "first street".
	DB.First(&freshUser.BillingAddress, freshUser.BillingAddressID)
	if freshUser.BillingAddress.Address1 != address1 {
		t.Errorf("Expected user's BillingAddress.Address1=%s to remain unchanged after UpdateColumns invocation, but BillingAddress.Address1=%s", address1, freshUser.BillingAddress.Address1)
	}
}

func TestUpdatesWithBlankValues(t *testing.T) {
	product := Product{Code: "product1", Price: 10}
	DB.Save(&product)

	DB.Model(&Product{Id: product.Id}).Updates(&Product{Price: 100})

	var product1 Product
	DB.First(&product1, product.Id)

	if product1.Code != "product1" || product1.Price != 100 {
		t.Errorf("product's code should not be updated")
	}
}

type ElementWithIgnoredField struct {
	Id           int64
	Value        string
	IgnoredField int64 `sql:"-"`
}

func (e ElementWithIgnoredField) TableName() string {
	return "element_with_ignored_field"
}

func TestUpdatesTableWithIgnoredValues(t *testing.T) {
	elem := ElementWithIgnoredField{Value: "foo", IgnoredField: 10}
	DB.Save(&elem)

	DB.Table(elem.TableName()).
		Where("id = ?", elem.Id).
		// DB.Model(&ElementWithIgnoredField{Id: elem.Id}).
		Updates(&ElementWithIgnoredField{Value: "bar", IgnoredField: 100})

	var elem1 ElementWithIgnoredField
	err := DB.First(&elem1, elem.Id).Error
	if err != nil {
		t.Errorf("error getting an element from database: %s", err.Error())
	}

	if elem1.IgnoredField != 0 {
		t.Errorf("element's ignored field should not be updated")
	}
}

func TestUpdateDecodeVirtualAttributes(t *testing.T) {
	var user = User{
		Name:     "jinzhu",
		IgnoreMe: 88,
	}

	DB.Save(&user)

	DB.Model(&user).Updates(User{Name: "jinzhu2", IgnoreMe: 100})

	if user.IgnoreMe != 100 {
		t.Errorf("should decode virtual attributes to struct, so it could be used in callbacks")
	}
}
