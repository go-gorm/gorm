package tests_test

import (
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func TestAssociationForBelongsTo(t *testing.T) {
	var user = *GetUser("belongs-to", Config{Company: true, Manager: true})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	CheckUser(t, user, user)

	// Find
	var user2 User
	DB.Find(&user2, "id = ?", user.ID)
	DB.Model(&user2).Association("Company").Find(&user2.Company)
	user2.Manager = &User{}
	DB.Model(&user2).Association("Manager").Find(user2.Manager)
	CheckUser(t, user2, user)

	// Count
	if count := DB.Model(&user).Association("Company").Count(); count != 1 {
		t.Errorf("invalid company count, got %v", count)
	}

	if count := DB.Model(&user).Association("Manager").Count(); count != 1 {
		t.Errorf("invalid manager count, got %v", count)
	}

	// Append
	var company = Company{Name: "company-belongs-to-append"}
	var manager = GetUser("manager-belongs-to-append", Config{})

	if err := DB.Model(&user2).Association("Company").Append(&company); err != nil {
		t.Fatalf("Error happened when append Company, got %v", err)
	}

	if company.ID == 0 {
		t.Fatalf("Company's ID should be created")
	}

	if err := DB.Model(&user2).Association("Manager").Append(manager); err != nil {
		t.Fatalf("Error happened when append Manager, got %v", err)
	}

	if manager.ID == 0 {
		t.Fatalf("Manager's ID should be created")
	}

	user.Company = company
	user.Manager = manager
	user.CompanyID = &company.ID
	user.ManagerID = &manager.ID
	CheckUser(t, user2, user)

	// Replace
	var company2 = Company{Name: "company-belongs-to-replace"}
	var manager2 = GetUser("manager-belongs-to-replace", Config{})

	if err := DB.Model(&user2).Association("Company").Replace(&company2); err != nil {
		t.Fatalf("Error happened when replace Company, got %v", err)
	}

	if company2.ID == 0 {
		t.Fatalf("Company's ID should be created")
	}

	if err := DB.Model(&user2).Association("Manager").Replace(manager2); err != nil {
		t.Fatalf("Error happened when replace Manager, got %v", err)
	}

	if manager2.ID == 0 {
		t.Fatalf("Manager's ID should be created")
	}

	user.Company = company2
	user.Manager = manager2
	user.CompanyID = &company2.ID
	user.ManagerID = &manager2.ID
	CheckUser(t, user2, user)

	// Delete
	if err := DB.Model(&user2).Association("Company").Delete(&Company{}); err != nil {
		t.Fatalf("Error happened when delete Company, got %v", err)
	}

	if count := DB.Model(&user2).Association("Company").Count(); count != 1 {
		t.Errorf("Invalid company count after delete non-existing association, got %v", count)
	}

	if err := DB.Model(&user2).Association("Company").Delete(&company2); err != nil {
		t.Fatalf("Error happened when delete Company, got %v", err)
	}

	if count := DB.Model(&user2).Association("Company").Count(); count != 0 {
		t.Errorf("Invalid company count after delete, got %v", count)
	}

	if err := DB.Model(&user2).Association("Manager").Delete(&User{}); err != nil {
		t.Fatalf("Error happened when delete Manager, got %v", err)
	}

	if count := DB.Model(&user2).Association("Manager").Count(); count != 1 {
		t.Errorf("Invalid manager count after delete non-existing association, got %v", count)
	}

	if err := DB.Model(&user2).Association("Manager").Delete(manager2); err != nil {
		t.Fatalf("Error happened when delete Manager, got %v", err)
	}

	if count := DB.Model(&user2).Association("Manager").Count(); count != 0 {
		t.Errorf("Invalid manager count after delete, got %v", count)
	}

	// Prepare Data
	if err := DB.Model(&user2).Association("Company").Append(&company); err != nil {
		t.Fatalf("Error happened when append Company, got %v", err)
	}

	if err := DB.Model(&user2).Association("Manager").Append(manager); err != nil {
		t.Fatalf("Error happened when append Manager, got %v", err)
	}

	if count := DB.Model(&user2).Association("Company").Count(); count != 1 {
		t.Errorf("Invalid company count after append, got %v", count)
	}

	if count := DB.Model(&user2).Association("Manager").Count(); count != 1 {
		t.Errorf("Invalid manager count after append, got %v", count)
	}

	// Clear
	if err := DB.Model(&user2).Association("Company").Clear(); err != nil {
		t.Errorf("Error happened when clear Company, got %v", err)
	}

	if err := DB.Model(&user2).Association("Manager").Clear(); err != nil {
		t.Errorf("Error happened when clear Manager, got %v", err)
	}

	if count := DB.Model(&user2).Association("Company").Count(); count != 0 {
		t.Errorf("Invalid company count after clear, got %v", count)
	}

	if count := DB.Model(&user2).Association("Manager").Count(); count != 0 {
		t.Errorf("Invalid manager count after clear, got %v", count)
	}
}
