package tests_test

import (
	"context"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	. "gorm.io/gorm/utils/tests"
)

// Test Set + Create with Association OpCreate operation using real database
func TestClauseAssociationSetCreateWithOpCreate(t *testing.T) {
	ctx := context.Background()

	// First create a user with Set + Create
	err := gorm.G[User](DB).Set(
		clause.Assignment{Column: clause.Column{Name: "name"}, Value: "TestClauseAssociationSetCreateWithOpCreate"},
		clause.Assignment{Column: clause.Column{Name: "age"}, Value: 25},
	).Create(ctx)
	if err != nil {
		t.Fatalf("Set Create failed: %v", err)
	}

	// Find the created user
	var user User
	if err := DB.Where("name = ?", "TestClauseAssociationSetCreateWithOpCreate").First(&user).Error; err != nil {
		t.Fatalf("failed to find created user: %v", err)
	}

	// Test Set + Update with Association OpCreate
	assocOp := clause.Association{
		Association: "Pets",
		Type:        clause.OpCreate,
		Set: []clause.Assignment{
			{Column: clause.Column{Name: "name"}, Value: "test-pet"},
		},
	}

	rows, err := gorm.G[User](DB).
		Where("id = ?", user.ID).
		Set(assocOp).
		Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with association failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	// Verify the association was created using real database query
	AssertAssociationCount(t, &user, "Pets", 1, "after Set Update with association")
}

// Test Set + Update with Association OpCreate operation using real database
func TestClauseAssociationSetUpdateWithOpCreate(t *testing.T) {
	ctx := context.Background()

	// Create a user with a pet first using real database
	user := User{Name: "TestClauseAssociationSetUpdateWithOpCreate", Age: 25}
	user.Pets = []*Pet{{Name: "original-pet"}}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user with pet: %v", err)
	}

	// Verify initial state using real database query
	AssertAssociationCount(t, user, "Pets", 1, "before update")

	// Test Set + Update with Association OpCreate
	assocOp := clause.Association{
		Association: "Pets",
		Type:        clause.OpCreate,
		Set: []clause.Assignment{
			{Column: clause.Column{Name: "name"}, Value: "new-pet"},
		},
	}

	rows, err := gorm.G[User](DB).
		Where("id = ?", user.ID).
		Set(assocOp).
		Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with association failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	// Verify the association was updated using real database query
	var updatedUser User
	if err := DB.Preload("Pets").Where("id = ?", user.ID).First(&updatedUser).Error; err != nil {
		t.Fatalf("failed to find updated user: %v", err)
	}

	if len(updatedUser.Pets) != 2 {
		t.Fatalf("expected 2 pets, got %d", len(updatedUser.Pets))
	}

	petNames := make(map[string]bool)
	for _, pet := range updatedUser.Pets {
		petNames[pet.Name] = true
	}

	if !petNames["original-pet"] {
		t.Error("original pet not found")
	}

	if !petNames["new-pet"] {
		t.Error("new pet not found")
	}
}

// Test Set + Create with multiple associations using real database
func TestClauseAssociationSetCreateWithMultipleAssociations(t *testing.T) {
	ctx := context.Background()

	// First create a user with Set + Create using real database
	err := gorm.G[User](DB).Set(
		clause.Assignment{Column: clause.Column{Name: "name"}, Value: "TestClauseAssociationSetCreateWithMultipleAssociations"},
		clause.Assignment{Column: clause.Column{Name: "age"}, Value: 25},
	).Create(ctx)
	if err != nil {
		t.Fatalf("Set Create failed: %v", err)
	}

	// Find the created user using real database query
	var user User
	if err := DB.Where("name = ?", "TestClauseAssociationSetCreateWithMultipleAssociations").First(&user).Error; err != nil {
		t.Fatalf("failed to find created user: %v", err)
	}

	// Test Set + Update with multiple association operations
	assocOp1 := clause.Association{
		Association: "Pets",
		Type:        clause.OpCreate,
		Set: []clause.Assignment{
			{Column: clause.Column{Name: "name"}, Value: "test-pet-1"},
		},
	}

	assocOp2 := clause.Association{
		Association: "Toys",
		Type:        clause.OpCreate,
		Set: []clause.Assignment{
			{Column: clause.Column{Name: "name"}, Value: "test-toy-1"},
		},
	}

	rows, err := gorm.G[User](DB).
		Where("id = ?", user.ID).
		Set(assocOp1, assocOp2).
		Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with multiple associations failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	// Verify both associations were created using real database queries
	AssertAssociationCount(t, &user, "Pets", 1, "after Set Update with multiple associations")
	AssertAssociationCount(t, &user, "Toys", 1, "after Set Update with multiple associations")
}

// Test Set + Update with multiple associations using real database
func TestClauseAssociationSetUpdateWithMultipleAssociations(t *testing.T) {
	ctx := context.Background()

	// Create a user with initial associations using real database
	user := User{Name: "TestClauseAssociationSetUpdateWithMultipleAssociations", Age: 25}
	user.Pets = []*Pet{{Name: "original-pet"}}
	user.Toys = []Toy{{Name: "original-toy"}}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user with associations: %v", err)
	}

	// Verify initial state using real database queries
	AssertAssociationCount(t, user, "Pets", 1, "before update")
	AssertAssociationCount(t, user, "Toys", 1, "before update")

	// Test Set + Update with multiple association operations
	assocOp1 := clause.Association{
		Association: "Pets",
		Type:        clause.OpCreate,
		Set: []clause.Assignment{
			{Column: clause.Column{Name: "name"}, Value: "new-pet"},
		},
	}

	assocOp2 := clause.Association{
		Association: "Toys",
		Type:        clause.OpCreate,
		Set: []clause.Assignment{
			{Column: clause.Column{Name: "name"}, Value: "new-toy"},
		},
	}

	rows, err := gorm.G[User](DB).
		Where("id = ?", user.ID).
		Set(assocOp1, assocOp2).
		Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with multiple associations failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	// Verify both associations were updated using real database queries
	var updatedUser User
	if err := DB.Preload("Pets").Preload("Toys").Where("id = ?", user.ID).First(&updatedUser).Error; err != nil {
		t.Fatalf("failed to find updated user: %v", err)
	}

	if len(updatedUser.Pets) != 2 {
		t.Fatalf("expected 2 pets, got %d", len(updatedUser.Pets))
	}

	if len(updatedUser.Toys) != 2 {
		t.Fatalf("expected 2 toys, got %d", len(updatedUser.Toys))
	}
}

// Test Set + Update with Association OpUnlink operation using real database
func TestClauseAssociationSetUpdateWithOpUnlink(t *testing.T) {
	ctx := context.Background()

	// Create a user with pets using real database
	user := User{Name: "TestClauseAssociationSetUpdateWithOpUnlink", Age: 25}
	user.Pets = []*Pet{{Name: "pet-to-unlink"}, {Name: "pet-to-keep"}}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user with pets: %v", err)
	}

	// Verify initial state using real database query
	AssertAssociationCount(t, user, "Pets", 2, "before unlink")

	// Get the pet to unlink using real database query
	var petToUnlink Pet
	if err := DB.Where("name = ?", "pet-to-unlink").First(&petToUnlink).Error; err != nil {
		t.Fatalf("failed to find pet to unlink: %v", err)
	}

	// Test Set + Update with Association OpUnlink
	assocOp := clause.Association{
		Association: "Pets",
		Type:        clause.OpUnlink,
		Conditions: []clause.Expression{
			clause.Eq{Column: clause.Column{Name: "id"}, Value: petToUnlink.ID},
		},
	}

	rows, err := gorm.G[User](DB).
		Where("id = ?", user.ID).
		Set(assocOp).
		Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with association unlink failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	// Verify only one pet remains using real database query
	var updatedUser User
	if err := DB.Preload("Pets").Where("id = ?", user.ID).First(&updatedUser).Error; err != nil {
		t.Fatalf("failed to find updated user: %v", err)
	}

	if len(updatedUser.Pets) != 1 {
		t.Fatalf("expected 1 pet after unlink, got %d", len(updatedUser.Pets))
	}

	if updatedUser.Pets[0].Name != "pet-to-keep" {
		t.Errorf("expected pet-to-keep, got %s", updatedUser.Pets[0].Name)
	}

	// Verify the unlinked pet still exists in the database using real database query
	var count int64
	if err := DB.Model(&Pet{}).Where("id = ?", petToUnlink.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count pet: %v", err)
	}
	if count != 1 {
		t.Error("unlinked pet should still exist in database")
	}
}

// Test Set + Update with Association OpCreateValues operation using real database
func TestClauseAssociationSetUpdateWithOpCreateValues(t *testing.T) {
	ctx := context.Background()

	// Create a user first using real database
	user := User{Name: "TestClauseAssociationSetUpdateWithOpCreateValues", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a pet object
	newPet := Pet{Name: "created-pet"}

	// Test Set + Update with Association OpCreateValues
	assocOp := clause.Association{
		Association: "Pets",
		Type:        clause.OpCreateValues,
		Values:      []interface{}{&newPet},
	}

	rows, err := gorm.G[User](DB).
		Where("id = ?", user.ID).
		Set(assocOp).
		Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with association create values failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	// Verify the pet was created and associated using real database query
	var updatedUser User
	if err := DB.Preload("Pets").Where("id = ?", user.ID).First(&updatedUser).Error; err != nil {
		t.Fatalf("failed to find updated user: %v", err)
	}

	if len(updatedUser.Pets) != 1 {
		t.Fatalf("expected 1 pet, got %d", len(updatedUser.Pets))
	}

	if updatedUser.Pets[0].Name != "created-pet" {
		t.Errorf("expected created-pet, got %s", updatedUser.Pets[0].Name)
	}
}

// Test Set + Create with many-to-many associations using real database
func TestClauseAssociationSetCreateWithManyToMany(t *testing.T) {
	ctx := context.Background()

	// Create a user first using real database
	user := User{Name: "TestClauseAssociationSetCreateWithManyToMany", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create languages using real database
	langs := []Language{
		{Code: "en", Name: "English"},
		{Code: "fr", Name: "French"},
	}
	for _, lang := range langs {
		DB.FirstOrCreate(&lang, "code = ?", lang.Code)
	}

	// Test Set + Update with many-to-many association
	assocOp := clause.Association{
		Association: "Languages",
		Type:        clause.OpCreateValues,
		Values:      []interface{}{langs[0], langs[1]},
	}

	rows, err := gorm.G[User](DB).
		Where("id = ?", user.ID).
		Set(assocOp).
		Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with many-to-many association failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	// Verify the languages were associated using real database query
	var updatedUser User
	if err := DB.Preload("Languages").Where("id = ?", user.ID).First(&updatedUser).Error; err != nil {
		t.Fatalf("failed to find updated user: %v", err)
	}

	if len(updatedUser.Languages) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(updatedUser.Languages))
	}
}

// Test Set + Create with belongs-to associations using real database
func TestClauseAssociationSetCreateWithBelongsTo(t *testing.T) {
	ctx := context.Background()

	// Create a company first using real database
	company := Company{Name: "Test Company"}
	if err := DB.Create(&company).Error; err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	// Test Set + Create with belongs-to association using field assignment
	err := gorm.G[User](DB).Set(
		clause.Assignment{Column: clause.Column{Name: "name"}, Value: "TestClauseAssociationSetCreateWithBelongsTo"},
		clause.Assignment{Column: clause.Column{Name: "age"}, Value: 25},
		clause.Assignment{Column: clause.Column{Name: "company_id"}, Value: company.ID},
	).Create(ctx)
	if err != nil {
		t.Fatalf("Set Create with belongs-to association failed: %v", err)
	}

	// Verify the user was created with company association using real database query
	var newUser User
	if err := DB.Preload("Company").Where("name = ?", "TestClauseAssociationSetCreateWithBelongsTo").First(&newUser).Error; err != nil {
		t.Fatalf("failed to find created user: %v", err)
	}

	if newUser.Company.ID != company.ID {
		t.Errorf("expected company ID %d, got %d", company.ID, newUser.Company.ID)
	}

	if newUser.Company.Name != company.Name {
		t.Errorf("expected company name %s, got %s", company.Name, newUser.Company.Name)
	}
}