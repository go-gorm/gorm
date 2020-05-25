package tests_test

import (
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func AssertAssociationCount(t *testing.T, data interface{}, name string, result int64, reason string) {
	if count := DB.Model(data).Association(name).Count(); count != result {
		t.Errorf("invalid %v count %v, expects: %v got %v", name, reason, result, count)
	}

	var newUser User
	if user, ok := data.(User); ok {
		DB.Find(&newUser, "id = ?", user.ID)
	} else if user, ok := data.(*User); ok {
		DB.Find(&newUser, "id = ?", user.ID)
	}

	if newUser.ID != 0 {
		if count := DB.Model(&newUser).Association(name).Count(); count != result {
			t.Errorf("invalid %v count %v, expects: %v got %v", name, reason, result, count)
		}
	}
}

func TestBelongsToAssociation(t *testing.T) {
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
	AssertAssociationCount(t, user, "Company", 1, "")
	AssertAssociationCount(t, user, "Manager", 1, "")

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

	AssertAssociationCount(t, user2, "Company", 1, "AfterAppend")
	AssertAssociationCount(t, user2, "Manager", 1, "AfterAppend")

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

	AssertAssociationCount(t, user2, "Company", 1, "AfterReplace")
	AssertAssociationCount(t, user2, "Manager", 1, "AfterReplace")

	// Delete
	if err := DB.Model(&user2).Association("Company").Delete(&Company{}); err != nil {
		t.Fatalf("Error happened when delete Company, got %v", err)
	}
	AssertAssociationCount(t, user2, "Company", 1, "after delete non-existing data")

	if err := DB.Model(&user2).Association("Company").Delete(&company2); err != nil {
		t.Fatalf("Error happened when delete Company, got %v", err)
	}
	AssertAssociationCount(t, user2, "Company", 0, "after delete")

	if err := DB.Model(&user2).Association("Manager").Delete(&User{}); err != nil {
		t.Fatalf("Error happened when delete Manager, got %v", err)
	}
	AssertAssociationCount(t, user2, "Manager", 1, "after delete non-existing data")

	if err := DB.Model(&user2).Association("Manager").Delete(manager2); err != nil {
		t.Fatalf("Error happened when delete Manager, got %v", err)
	}
	AssertAssociationCount(t, user2, "Manager", 0, "after delete")

	// Prepare Data for Clear
	if err := DB.Model(&user2).Association("Company").Append(&company); err != nil {
		t.Fatalf("Error happened when append Company, got %v", err)
	}

	if err := DB.Model(&user2).Association("Manager").Append(manager); err != nil {
		t.Fatalf("Error happened when append Manager, got %v", err)
	}

	AssertAssociationCount(t, user2, "Company", 1, "after prepare data")
	AssertAssociationCount(t, user2, "Manager", 1, "after prepare data")

	// Clear
	if err := DB.Model(&user2).Association("Company").Clear(); err != nil {
		t.Errorf("Error happened when clear Company, got %v", err)
	}

	if err := DB.Model(&user2).Association("Manager").Clear(); err != nil {
		t.Errorf("Error happened when clear Manager, got %v", err)
	}

	AssertAssociationCount(t, user2, "Company", 0, "after clear")
	AssertAssociationCount(t, user2, "Manager", 0, "after clear")
}

func TestBelongsToAssociationForSlice(t *testing.T) {
	var users = []User{
		*GetUser("slice-belongs-to-1", Config{Company: true, Manager: true}),
		*GetUser("slice-belongs-to-2", Config{Company: true, Manager: false}),
		*GetUser("slice-belongs-to-3", Config{Company: true, Manager: true}),
	}

	DB.Create(&users)

	AssertAssociationCount(t, users, "Company", 3, "")
	AssertAssociationCount(t, users, "Manager", 2, "")

	// Find
	var companies []Company
	if DB.Model(&users).Association("Company").Find(&companies); len(companies) != 3 {
		t.Errorf("companies count should be %v, but got %v", 3, len(companies))
	}

	var managers []User
	if DB.Model(&users).Association("Manager").Find(&managers); len(managers) != 2 {
		t.Errorf("managers count should be %v, but got %v", 2, len(managers))
	}

	// Append
	DB.Model(&users).Association("Company").Append(
		&Company{Name: "company-slice-append-1"},
		&Company{Name: "company-slice-append-2"},
		&Company{Name: "company-slice-append-3"},
	)

	AssertAssociationCount(t, users, "Company", 3, "After Append")

	DB.Model(&users).Association("Manager").Append(
		GetUser("manager-slice-belongs-to-1", Config{}),
		GetUser("manager-slice-belongs-to-2", Config{}),
		GetUser("manager-slice-belongs-to-3", Config{}),
	)
	AssertAssociationCount(t, users, "Manager", 3, "After Append")

	if err := DB.Model(&users).Association("Manager").Append(
		GetUser("manager-slice-belongs-to-test-1", Config{}),
	).Error; err == nil {
		t.Errorf("unmatched length when update user's manager")
	}

	// Replace -> same as append

	// Delete
	if err := DB.Model(&users).Association("Company").Delete(&users[0].Company); err != nil {
		t.Errorf("no error should happend when deleting company, but got %v", err)
	}

	if users[0].CompanyID != nil || users[0].Company.ID != 0 {
		t.Errorf("users[0]'s company should be deleted'")
	}

	AssertAssociationCount(t, users, "Company", 2, "After Delete")

	// Clear
	DB.Model(&users).Association("Company").Clear()
	AssertAssociationCount(t, users, "Company", 0, "After Clear")

	DB.Model(&users).Association("Manager").Clear()
	AssertAssociationCount(t, users, "Manager", 0, "After Clear")

	// shared company
	company := Company{Name: "shared"}
	if err := DB.Model(&users[0]).Association("Company").Append(&company); err != nil {
		t.Errorf("Error happened when append company to user, got %v", err)
	}

	if err := DB.Model(&users[1]).Association("Company").Append(&company); err != nil {
		t.Errorf("Error happened when append company to user, got %v", err)
	}

	if users[0].CompanyID == nil || users[1].CompanyID == nil || *users[0].CompanyID != *users[1].CompanyID {
		t.Errorf("user's company id should exists and equal, but its: %v, %v", users[0].CompanyID, users[1].CompanyID)
	}

	DB.Model(&users[0]).Association("Company").Delete(&company)
	AssertAssociationCount(t, users[0], "Company", 0, "After Delete")
	AssertAssociationCount(t, users[1], "Company", 1, "After other user Delete")
}

func TestHasOneAssociation(t *testing.T) {
	var user = *GetUser("hasone", Config{Account: true})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	CheckUser(t, user, user)

	// Find
	var user2 User
	DB.Find(&user2, "id = ?", user.ID)
	DB.Model(&user2).Association("Account").Find(&user2.Account)
	CheckUser(t, user2, user)

	// Count
	AssertAssociationCount(t, user, "Account", 1, "")

	// Append
	var account = Account{Number: "account-has-one-append"}

	if err := DB.Model(&user2).Association("Account").Append(&account); err != nil {
		t.Fatalf("Error happened when append account, got %v", err)
	}

	if account.ID == 0 {
		t.Fatalf("Account's ID should be created")
	}

	user.Account = account
	CheckUser(t, user2, user)

	AssertAssociationCount(t, user, "Account", 1, "AfterAppend")

	// Replace
	var account2 = Account{Number: "account-has-one-replace"}

	if err := DB.Model(&user2).Association("Account").Replace(&account2); err != nil {
		t.Fatalf("Error happened when append Account, got %v", err)
	}

	if account2.ID == 0 {
		t.Fatalf("account2's ID should be created")
	}

	user.Account = account2
	CheckUser(t, user2, user)

	AssertAssociationCount(t, user2, "Account", 1, "AfterReplace")

	// Delete
	if err := DB.Model(&user2).Association("Account").Delete(&Account{}); err != nil {
		t.Fatalf("Error happened when delete account, got %v", err)
	}
	AssertAssociationCount(t, user2, "Account", 1, "after delete non-existing data")

	if err := DB.Model(&user2).Association("Account").Delete(&account2); err != nil {
		t.Fatalf("Error happened when delete Account, got %v", err)
	}
	AssertAssociationCount(t, user2, "Account", 0, "after delete")

	// Prepare Data for Clear
	if err := DB.Model(&user2).Association("Account").Append(&account); err != nil {
		t.Fatalf("Error happened when append Account, got %v", err)
	}

	AssertAssociationCount(t, user2, "Account", 1, "after prepare data")

	// Clear
	if err := DB.Model(&user2).Association("Account").Clear(); err != nil {
		t.Errorf("Error happened when clear Account, got %v", err)
	}

	AssertAssociationCount(t, user2, "Account", 0, "after clear")
}

func TestHasOneAssociationForSlice(t *testing.T) {
	var users = []User{
		*GetUser("slice-hasone-1", Config{Account: true}),
		*GetUser("slice-hasone-2", Config{Account: false}),
		*GetUser("slice-hasone-3", Config{Account: true}),
	}

	DB.Create(&users)

	// Count
	AssertAssociationCount(t, users, "Account", 2, "")

	// Find
	var accounts []Account
	if DB.Model(&users).Association("Account").Find(&accounts); len(accounts) != 2 {
		t.Errorf("accounts count should be %v, but got %v", 3, len(accounts))
	}

	// Append
	DB.Model(&users).Association("Account").Append(
		&Account{Number: "account-slice-append-1"},
		&Account{Number: "account-slice-append-2"},
		&Account{Number: "account-slice-append-3"},
	)

	AssertAssociationCount(t, users, "Account", 3, "After Append")

	// Replace -> same as append

	// Delete
	if err := DB.Model(&users).Association("Account").Delete(&users[0].Account); err != nil {
		t.Errorf("no error should happend when deleting account, but got %v", err)
	}

	AssertAssociationCount(t, users, "Account", 2, "after delete")

	// Clear
	DB.Model(&users).Association("Account").Clear()
	AssertAssociationCount(t, users, "Account", 0, "After Clear")
}

func TestPolymorphicHasOneAssociation(t *testing.T) {
	var pet = Pet{Name: "hasone", Toy: Toy{Name: "toy-has-one"}}

	if err := DB.Create(&pet).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	CheckPet(t, pet, pet)

	// Find
	var pet2 Pet
	DB.Find(&pet2, "id = ?", pet.ID)
	DB.Model(&pet2).Association("Toy").Find(&pet2.Toy)
	CheckPet(t, pet2, pet)

	// Count
	AssertAssociationCount(t, pet, "Toy", 1, "")

	// Append
	var toy = Toy{Name: "toy-has-one-append"}

	if err := DB.Model(&pet2).Association("Toy").Append(&toy); err != nil {
		t.Fatalf("Error happened when append toy, got %v", err)
	}

	if toy.ID == 0 {
		t.Fatalf("Toy's ID should be created")
	}

	pet.Toy = toy
	CheckPet(t, pet2, pet)

	AssertAssociationCount(t, pet, "Toy", 1, "AfterAppend")

	// Replace
	var toy2 = Toy{Name: "toy-has-one-replace"}

	if err := DB.Model(&pet2).Association("Toy").Replace(&toy2); err != nil {
		t.Fatalf("Error happened when append Toy, got %v", err)
	}

	if toy2.ID == 0 {
		t.Fatalf("toy2's ID should be created")
	}

	pet.Toy = toy2
	CheckPet(t, pet2, pet)

	AssertAssociationCount(t, pet2, "Toy", 1, "AfterReplace")

	// Delete
	if err := DB.Model(&pet2).Association("Toy").Delete(&Toy{}); err != nil {
		t.Fatalf("Error happened when delete toy, got %v", err)
	}
	AssertAssociationCount(t, pet2, "Toy", 1, "after delete non-existing data")

	if err := DB.Model(&pet2).Association("Toy").Delete(&toy2); err != nil {
		t.Fatalf("Error happened when delete Toy, got %v", err)
	}
	AssertAssociationCount(t, pet2, "Toy", 0, "after delete")

	// Prepare Data for Clear
	if err := DB.Model(&pet2).Association("Toy").Append(&toy); err != nil {
		t.Fatalf("Error happened when append Toy, got %v", err)
	}

	AssertAssociationCount(t, pet2, "Toy", 1, "after prepare data")

	// Clear
	if err := DB.Model(&pet2).Association("Toy").Clear(); err != nil {
		t.Errorf("Error happened when clear Toy, got %v", err)
	}

	AssertAssociationCount(t, pet2, "Toy", 0, "after clear")
}

func TestPolymorphicHasOneAssociationForSlice(t *testing.T) {
	var pets = []Pet{
		{Name: "hasone-1", Toy: Toy{Name: "toy-has-one"}},
		{Name: "hasone-2", Toy: Toy{}},
		{Name: "hasone-3", Toy: Toy{Name: "toy-has-one"}},
	}

	DB.Create(&pets)

	// Count
	AssertAssociationCount(t, pets, "Toy", 2, "")

	// Find
	var toys []Toy
	if DB.Model(&pets).Association("Toy").Find(&toys); len(toys) != 2 {
		t.Errorf("toys count should be %v, but got %v", 3, len(toys))
	}

	// Append
	DB.Model(&pets).Association("Toy").Append(
		&Toy{Name: "toy-slice-append-1"},
		&Toy{Name: "toy-slice-append-2"},
		&Toy{Name: "toy-slice-append-3"},
	)

	AssertAssociationCount(t, pets, "Toy", 3, "After Append")

	// Replace -> same as append

	// Delete
	if err := DB.Model(&pets).Association("Toy").Delete(&pets[0].Toy); err != nil {
		t.Errorf("no error should happend when deleting toy, but got %v", err)
	}

	AssertAssociationCount(t, pets, "Toy", 2, "after delete")

	// Clear
	DB.Model(&pets).Association("Toy").Clear()
	AssertAssociationCount(t, pets, "Toy", 0, "After Clear")
}

func TestHasManyAssociation(t *testing.T) {
	var user = *GetUser("hasmany", Config{Pets: 2})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	CheckUser(t, user, user)

	// Find
	var user2 User
	DB.Find(&user2, "id = ?", user.ID)
	DB.Model(&user2).Association("Pets").Find(&user2.Pets)
	CheckUser(t, user2, user)

	// Count
	AssertAssociationCount(t, user, "Pets", 2, "")

	// Append
	var pet = Pet{Name: "pet-has-many-append"}

	if err := DB.Model(&user2).Association("Pets").Append(&pet); err != nil {
		t.Fatalf("Error happened when append account, got %v", err)
	}

	if pet.ID == 0 {
		t.Fatalf("Pet's ID should be created")
	}

	user.Pets = append(user.Pets, &pet)
	CheckUser(t, user2, user)

	AssertAssociationCount(t, user, "Pets", 3, "AfterAppend")

	var pets = []Pet{{Name: "pet-has-many-append-1-1"}, {Name: "pet-has-many-append-1-1"}}

	if err := DB.Model(&user2).Association("Pets").Append(&pets); err != nil {
		t.Fatalf("Error happened when append pet, got %v", err)
	}

	for _, pet := range pets {
		var pet = pet
		if pet.ID == 0 {
			t.Fatalf("Pet's ID should be created")
		}

		user.Pets = append(user.Pets, &pet)
	}

	CheckUser(t, user2, user)

	AssertAssociationCount(t, user, "Pets", 5, "AfterAppendSlice")

	// Replace
	var pet2 = Pet{Name: "pet-has-many-replace"}

	if err := DB.Model(&user2).Association("Pets").Replace(&pet2); err != nil {
		t.Fatalf("Error happened when append pet, got %v", err)
	}

	if pet2.ID == 0 {
		t.Fatalf("pet2's ID should be created")
	}

	user.Pets = []*Pet{&pet2}
	CheckUser(t, user2, user)

	AssertAssociationCount(t, user2, "Pets", 1, "AfterReplace")

	// Delete
	if err := DB.Model(&user2).Association("Pets").Delete(&Pet{}); err != nil {
		t.Fatalf("Error happened when delete pet, got %v", err)
	}
	AssertAssociationCount(t, user2, "Pets", 1, "after delete non-existing data")

	if err := DB.Model(&user2).Association("Pets").Delete(&pet2); err != nil {
		t.Fatalf("Error happened when delete Pets, got %v", err)
	}
	AssertAssociationCount(t, user2, "Pets", 0, "after delete")

	// Prepare Data for Clear
	if err := DB.Model(&user2).Association("Pets").Append(&pet); err != nil {
		t.Fatalf("Error happened when append Pets, got %v", err)
	}

	AssertAssociationCount(t, user2, "Pets", 1, "after prepare data")

	// Clear
	if err := DB.Model(&user2).Association("Pets").Clear(); err != nil {
		t.Errorf("Error happened when clear Pets, got %v", err)
	}

	AssertAssociationCount(t, user2, "Pets", 0, "after clear")
}

func TestHasManyAssociationForSlice(t *testing.T) {
	var users = []User{
		*GetUser("slice-hasmany-1", Config{Pets: 2}),
		*GetUser("slice-hasmany-2", Config{Pets: 0}),
		*GetUser("slice-hasmany-3", Config{Pets: 4}),
	}

	DB.Create(&users)

	// Count
	AssertAssociationCount(t, users, "Pets", 6, "")

	// Find
	var pets []Pet
	if DB.Model(&users).Association("Pets").Find(&pets); len(pets) != 6 {
		t.Errorf("pets count should be %v, but got %v", 6, len(pets))
	}

	// Append
	DB.Model(&users).Association("Pets").Append(
		&Pet{Name: "pet-slice-append-1"},
		[]*Pet{{Name: "pet-slice-append-2-1"}, {Name: "pet-slice-append-2-2"}},
		&Pet{Name: "pet-slice-append-3"},
	)

	AssertAssociationCount(t, users, "Pets", 10, "After Append")

	// Replace -> same as append
	DB.Model(&users).Association("Pets").Replace(
		[]*Pet{{Name: "pet-slice-replace-1-1"}, {Name: "pet-slice-replace-1-2"}},
		[]*Pet{{Name: "pet-slice-replace-2-1"}, {Name: "pet-slice-replace-2-2"}},
		&Pet{Name: "pet-slice-replace-3"},
	)

	AssertAssociationCount(t, users, "Pets", 5, "After Append")

	// Delete
	if err := DB.Model(&users).Association("Pets").Delete(&users[2].Pets); err != nil {
		t.Errorf("no error should happend when deleting pet, but got %v", err)
	}

	AssertAssociationCount(t, users, "Pets", 4, "after delete")

	if err := DB.Model(&users).Association("Pets").Delete(users[0].Pets[0], users[1].Pets[1]); err != nil {
		t.Errorf("no error should happend when deleting pet, but got %v", err)
	}

	AssertAssociationCount(t, users, "Pets", 2, "after delete")

	// Clear
	DB.Model(&users).Association("Pets").Clear()
	AssertAssociationCount(t, users, "Pets", 0, "After Clear")
}

func TestPolymorphicHasManyAssociation(t *testing.T) {
	var user = *GetUser("hasmany", Config{Toys: 2})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	CheckUser(t, user, user)

	// Find
	var user2 User
	DB.Find(&user2, "id = ?", user.ID)
	DB.Model(&user2).Association("Toys").Find(&user2.Toys)
	CheckUser(t, user2, user)

	// Count
	AssertAssociationCount(t, user, "Toys", 2, "")

	// Append
	var toy = Toy{Name: "toy-has-many-append"}

	if err := DB.Model(&user2).Association("Toys").Append(&toy); err != nil {
		t.Fatalf("Error happened when append account, got %v", err)
	}
	return

	if toy.ID == 0 {
		t.Fatalf("Toy's ID should be created")
	}

	user.Toys = append(user.Toys, toy)
	CheckUser(t, user2, user)

	AssertAssociationCount(t, user, "Toys", 3, "AfterAppend")

	var toys = []Toy{{Name: "toy-has-many-append-1-1"}, {Name: "toy-has-many-append-1-1"}}

	if err := DB.Model(&user2).Association("Toys").Append(&toys); err != nil {
		t.Fatalf("Error happened when append toy, got %v", err)
	}

	for _, toy := range toys {
		var toy = toy
		if toy.ID == 0 {
			t.Fatalf("Toy's ID should be created")
		}

		user.Toys = append(user.Toys, toy)
	}

	CheckUser(t, user2, user)

	AssertAssociationCount(t, user, "Toys", 5, "AfterAppendSlice")

	// Replace
	var toy2 = Toy{Name: "toy-has-many-replace"}

	if err := DB.Model(&user2).Association("Toys").Replace(&toy2); err != nil {
		t.Fatalf("Error happened when append toy, got %v", err)
	}

	if toy2.ID == 0 {
		t.Fatalf("toy2's ID should be created")
	}

	user.Toys = []Toy{toy2}
	CheckUser(t, user2, user)

	AssertAssociationCount(t, user2, "Toys", 1, "AfterReplace")

	// Delete
	if err := DB.Model(&user2).Association("Toys").Delete(&Toy{}); err != nil {
		t.Fatalf("Error happened when delete toy, got %v", err)
	}
	AssertAssociationCount(t, user2, "Toys", 1, "after delete non-existing data")

	if err := DB.Model(&user2).Association("Toys").Delete(&toy2); err != nil {
		t.Fatalf("Error happened when delete Toys, got %v", err)
	}
	AssertAssociationCount(t, user2, "Toys", 0, "after delete")

	// Prepare Data for Clear
	if err := DB.Model(&user2).Association("Toys").Append(&toy); err != nil {
		t.Fatalf("Error happened when append Toys, got %v", err)
	}

	AssertAssociationCount(t, user2, "Toys", 1, "after prepare data")

	// Clear
	if err := DB.Model(&user2).Association("Toys").Clear(); err != nil {
		t.Errorf("Error happened when clear Toys, got %v", err)
	}

	AssertAssociationCount(t, user2, "Toys", 0, "after clear")
}

func TestPolymorphicHasManyAssociationForSlice(t *testing.T) {
	var users = []User{
		*GetUser("slice-hasmany-1", Config{Toys: 2}),
		*GetUser("slice-hasmany-2", Config{Toys: 0}),
		*GetUser("slice-hasmany-3", Config{Toys: 4}),
	}

	DB.Create(&users)

	// Count
	AssertAssociationCount(t, users, "Toys", 6, "")

	// Find
	var toys []Toy
	if DB.Model(&users).Association("Toys").Find(&toys); len(toys) != 6 {
		t.Errorf("toys count should be %v, but got %v", 6, len(toys))
	}

	// Append
	DB.Model(&users).Association("Toys").Append(
		&Toy{Name: "toy-slice-append-1"},
		[]Toy{{Name: "toy-slice-append-2-1"}, {Name: "toy-slice-append-2-2"}},
		&Toy{Name: "toy-slice-append-3"},
	)

	AssertAssociationCount(t, users, "Toys", 10, "After Append")

	// Replace -> same as append
	DB.Model(&users).Association("Toys").Replace(
		[]*Toy{{Name: "toy-slice-replace-1-1"}, {Name: "toy-slice-replace-1-2"}},
		[]*Toy{{Name: "toy-slice-replace-2-1"}, {Name: "toy-slice-replace-2-2"}},
		&Toy{Name: "toy-slice-replace-3"},
	)

	AssertAssociationCount(t, users, "Toys", 5, "After Append")

	// Delete
	if err := DB.Model(&users).Association("Toys").Delete(&users[2].Toys); err != nil {
		t.Errorf("no error should happend when deleting toy, but got %v", err)
	}

	AssertAssociationCount(t, users, "Toys", 4, "after delete")

	if err := DB.Model(&users).Association("Toys").Delete(users[0].Toys[0], users[1].Toys[1]); err != nil {
		t.Errorf("no error should happend when deleting toy, but got %v", err)
	}

	AssertAssociationCount(t, users, "Toys", 2, "after delete")

	// Clear
	DB.Model(&users).Association("Toys").Clear()
	AssertAssociationCount(t, users, "Toys", 0, "After Clear")
}
