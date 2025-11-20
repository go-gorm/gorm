package tests_test

import (
	"context"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	. "gorm.io/gorm/utils/tests"
)

// BelongsToCompany and BelongsToUser models for belongs to tests - using existing User and Company models

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

	rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with association failed: %v", err)
	}
	// Only association operations were executed; no row update is expected
	if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
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

	rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with association failed: %v", err)
	}
	// Only association operations were executed; no row update is expected
	if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
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

	rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp1, assocOp2).Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with multiple associations failed: %v", err)
	}
	// Only association operations were executed; no row update is expected
	if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
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

	rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp1, assocOp2).Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with multiple associations failed: %v", err)
	}
	// Only association operations were executed; no row update is expected
	if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
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

	rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with association unlink failed: %v", err)
	}
	// Only association operations were executed; no row update is expected
	if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
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

// Test Set + Update with Association OpCreate operation using real database
func TestClauseAssociationSetUpdateWithOpCreateValues(t *testing.T) {
	ctx := context.Background()

	// Create a user first using real database
	user := User{Name: "TestClauseAssociationSetUpdateWithOpCreate", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a pet object
	newPet := Pet{Name: "created-pet"}

	// Test Set + Update with Association OpCreate
	assocOp := clause.Association{
		Association: "Pets",
		Type:        clause.OpCreate,
		Values:      []interface{}{&newPet},
	}

	rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with association create values failed: %v", err)
	}
	// Only association operations were executed; no row update is expected
	if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
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
		Type:        clause.OpCreate,
		Values:      []interface{}{langs[0], langs[1]},
	}

	rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx)
	if err != nil {
		t.Fatalf("Set Update with many-to-many association failed: %v", err)
	}
	// Only association operations were executed; no row update is expected
	if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
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

// BelongsTo: create and assign company via OpCreate
func TestClauseAssociationSetUpdateBelongsToCreateValues(t *testing.T) {
	ctx := context.Background()

	user := User{Name: "TestClauseAssociationSetUpdateBelongsToCreateValues", Age: 26}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	assocOp := clause.Association{Association: "Company", Type: clause.OpCreate, Values: []interface{}{Company{Name: "Belongs-To-Co"}}}
	if rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx); err != nil {
		t.Fatalf("Set Update belongs-to create values failed: %v", err)
	} else if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
	}

	var got User
	if err := DB.Preload("Company").First(&got, user.ID).Error; err != nil {
		t.Fatalf("failed preload company: %v", err)
	}
	if got.Company.ID == 0 || got.Company.Name != "Belongs-To-Co" {
		t.Fatalf("expected Company assigned, got %+v", got.Company)
	}
}

// Mixed fields + association: update Age and create a pet together
func TestClauseAssociationSetUpdateMixedFieldAndAssociation(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdateMixed", Age: 20}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	assocOp := clause.Association{Association: "Pets", Type: clause.OpCreate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "mix-pet"}}}
	rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(
		assocOp,
		clause.Assignment{Column: clause.Column{Name: "age"}, Value: 30},
	).Update(ctx)
	if err != nil {
		t.Fatalf("Set Update mixed failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected for field update, got %d", rows)
	}

	var got User
	if err := DB.Preload("Pets").First(&got, user.ID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if got.Age != 30 {
		t.Fatalf("expected age 30, got %d", got.Age)
	}
	if len(got.Pets) != 1 || got.Pets[0].Name != "mix-pet" {
		t.Fatalf("expected pet created, got %+v", got.Pets)
	}
}

// HasOne unlink clears NamedPet
func TestClauseAssociationSetUpdateHasOneUnlink(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdateHasOneUnlink", Age: 25}
	user.NamedPet = &Pet{Name: "np"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	assocOp := clause.Association{Association: "NamedPet", Type: clause.OpUnlink}
	if rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx); err != nil {
		t.Fatalf("Set Update has-one unlink failed: %v", err)
	} else if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
	}

	var got User
	if err := DB.Preload("NamedPet").First(&got, user.ID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if got.NamedPet != nil {
		t.Fatalf("expected NamedPet cleared, got %+v", got.NamedPet)
	}
}

// Many-to-Many create with Set
func TestClauseAssociationSetUpdateManyToManyCreateWithSet(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdateMany2ManyCreateWithSet", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	assocOp := clause.Association{
		Association: "Languages", Type: clause.OpCreate,
		Set: []clause.Assignment{{Column: clause.Column{Name: "code"}, Value: "it"}, {Column: clause.Column{Name: "name"}, Value: "Italian"}},
	}
	if rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx); err != nil {
		t.Fatalf("Set Update many2many create with set failed: %v", err)
	} else if rows != 0 {
		t.Fatalf("expected 0 rows affected, got %d", rows)
	}

	AssertAssociationCount(t, user, "Languages", 1, "after create language")
}

// Many-to-Many clear
func TestClauseAssociationSetUpdateManyToManyClear(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdateMany2ManyClear", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	langs := []Language{{Code: "pt", Name: "Portuguese"}, {Code: "ru", Name: "Russian"}}
	for _, l := range langs {
		DB.FirstOrCreate(&l, "code = ?", l.Code)
	}
	if err := DB.Model(&user).Association("Languages").Append(&langs); err != nil {
		t.Fatalf("append: %v", err)
	}
	AssertAssociationCount(t, user, "Languages", 2, "before clear")

	assocOp := clause.Association{Association: "Languages", Type: clause.OpUnlink}
	if rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx); err != nil {
		t.Fatalf("Set Update many2many clear failed: %v", err)
	} else if rows != 0 {
		t.Fatalf("expected 0 rows affected, got %d", rows)
	}
	AssertAssociationCount(t, user, "Languages", 0, "after clear")
}

// Polymorphic Tools create and unlink
func TestClauseAssociationSetUpdatePolymorphicTools(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdatePolymorphicTools", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	createOp := clause.Association{Association: "Tools", Type: clause.OpCreate, Values: []interface{}{Tools{Name: "wrench"}}}
	if rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(createOp).Update(ctx); err != nil {
		t.Fatalf("create tools: %v", err)
	} else if rows != 0 {
		t.Fatalf("rows %d", rows)
	}
	AssertAssociationCount(t, user, "Tools", 1, "after create tool")

	unlinkOp := clause.Association{Association: "Tools", Type: clause.OpUnlink}
	if rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(unlinkOp).Update(ctx); err != nil {
		t.Fatalf("unlink tools: %v", err)
	} else if rows != 0 {
		t.Fatalf("rows %d", rows)
	}
	AssertAssociationCount(t, user, "Tools", 0, "after clear tools")
}

// Invalid association should return error
func TestClauseAssociationSetUpdateInvalidAssociation(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdateInvalidAssociation", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	assocOp := clause.Association{Association: "Invalid", Type: clause.OpCreate}
	if _, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx); err == nil {
		t.Fatalf("expected error for invalid association, got nil")
	}
}

// No owner matched; should be no-op
func TestClauseAssociationSetUpdateNoOwnerMatch(t *testing.T) {
	ctx := context.Background()
	assocOp := clause.Association{Association: "Pets", Type: clause.OpCreate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "won't-create"}}}
	if rows, err := gorm.G[User](DB).Where("id = ?", -1).Set(assocOp).Update(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if rows != 0 {
		t.Fatalf("expected 0 rows, got %d", rows)
	}
}

// OpDelete/OpUpdate should work for associations
func TestClauseAssociationSetUpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdateAndDelete", Age: 25}
	user.Pets = []*Pet{{Name: "before"}}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	AssertAssociationCount(t, user, "Pets", 1, "before update/delete")

	// Update pet name via OpUpdate
	updOp := clause.Association{Association: "Pets", Type: clause.OpUpdate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "x"}}}
	if _, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(updOp).Update(ctx); err != nil {
		t.Fatalf("OpUpdate failed: %v", err)
	}
	var got User
	if err := DB.Preload("Pets").First(&got, user.ID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if len(got.Pets) != 1 || got.Pets[0].Name != "x" {
		t.Fatalf("expected updated pet name, got %+v", got.Pets)
	}

	// Delete pets via OpDelete
	delOp := clause.Association{Association: "Pets", Type: clause.OpDelete}
	if _, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(delOp).Update(ctx); err != nil {
		t.Fatalf("OpDelete failed: %v", err)
	}
	AssertAssociationCount(t, user, "Pets", 0, "after delete")
}

// HasOne: update and delete NamedPet via OpUpdate/OpDelete
func TestClauseAssociationSetUpdateAndDeleteHasOne(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdateAndDeleteHasOne", Age: 25}
	user.NamedPet = &Pet{Name: "np-before"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	AssertAssociationCount(t, user, "NamedPet", 1, "before")

	upd := clause.Association{Association: "NamedPet", Type: clause.OpUpdate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "np-after"}}}
	if _, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(upd).Update(ctx); err != nil {
		t.Fatalf("OpUpdate has-one failed: %v", err)
	}
	var u1 User
	if err := DB.Preload("NamedPet").First(&u1, user.ID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if u1.NamedPet == nil || u1.NamedPet.Name != "np-after" {
		t.Fatalf("expected name updated, got %+v", u1.NamedPet)
	}

	del := clause.Association{Association: "NamedPet", Type: clause.OpDelete}
	if _, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(del).Update(ctx); err != nil {
		t.Fatalf("OpDelete has-one failed: %v", err)
	}
	AssertAssociationCount(t, user, "NamedPet", 0, "after delete")
}

// Many2Many append with map using Association API (regression for map support)
func TestAssociationMany2ManyAppendMap_GenericFile(t *testing.T) {
	user := User{Name: "AssocM2MAppendMapGeneric", Age: 28}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := DB.Model(&user).Association("Languages").Append(map[string]interface{}{
		"code": "gm2m_map_1", "name": "GMap1",
	}); err != nil {
		t.Fatalf("append map: %v", err)
	}
	AssertAssociationCount(t, user, "Languages", 1, "after append 1 map (generic file)")

	// Append more maps individually
	if err := DB.Model(&user).Association("Languages").Append(map[string]interface{}{"code": "gm2m_map_2", "name": "GMap2"}); err != nil {
		t.Fatalf("append map 2: %v", err)
	}
	if err := DB.Model(&user).Association("Languages").Append(map[string]interface{}{"code": "gm2m_map_3", "name": "GMap3"}); err != nil {
		t.Fatalf("append map 3: %v", err)
	}
	AssertAssociationCount(t, user, "Languages", 3, "after append 3 maps total (generic file)")
}

// BelongsTo: update and delete Company via OpUpdate/OpDelete
func TestClauseAssociationSetUpdateAndDeleteBelongsTo(t *testing.T) {
	ctx := context.Background()

	// Create company and user with company
	company := Company{Name: "Electronics"}
	if err := DB.Create(&company).Error; err != nil {
		t.Fatalf("create company: %v", err)
	}

	user := User{Name: "John", Age: 25, CompanyID: &company.ID}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Verify association exists
	AssertAssociationCount(t, &user, "Company", 1, "before")

	// Update company name via OpUpdate
	upd := clause.Association{Association: "Company", Type: clause.OpUpdate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "Electronics-New"}}}
	if _, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(upd).Update(ctx); err != nil {
		t.Fatalf("OpUpdate belongs-to failed: %v", err)
	}

	var u1 User
	if err := DB.Preload("Company").First(&u1, user.ID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}

	if u1.Company.ID == 0 || u1.Company.Name != "Electronics-New" {
		t.Fatalf("expected company updated, got %+v", u1.Company)
	}

	// Unlink company association via OpUnlink (instead of OpDelete which would try to delete the company record)
	unlink := clause.Association{Association: "Company", Type: clause.OpUnlink}
	if _, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(unlink).Update(ctx); err != nil {
		t.Fatalf("OpUnlink belongs-to failed: %v", err)
	}

	var u2 User
	if err := DB.Preload("Company").First(&u2, user.ID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}

	if u2.Company.ID != 0 {
		t.Fatalf("expected company association cleared due to unlink, got %+v", u2.Company)
	}
}

// Many2Many: update and delete via Set
func TestClauseAssociationSetUpdateAndDeleteMany2Many(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdateAndDeleteMany2Many", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	langs := []Language{{Code: "es", Name: "Spanish"}, {Code: "de", Name: "German"}}
	for _, l := range langs {
		DB.FirstOrCreate(&l, "code = ?", l.Code)
	}
	if err := DB.Model(&user).Association("Languages").Append(&langs); err != nil {
		t.Fatalf("append: %v", err)
	}
	AssertAssociationCount(t, user, "Languages", 2, "before")

	upd := clause.Association{Association: "Languages", Type: clause.OpUpdate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "Espanol"}}, Conditions: []clause.Expression{clause.Eq{Column: clause.Column{Name: "code"}, Value: "es"}}}
	if _, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(upd).Update(ctx); err != nil {
		t.Fatalf("OpUpdate m2m failed: %v", err)
	}
	var es Language
	if err := DB.First(&es, "code = ?", "es").Error; err != nil {
		t.Fatalf("load lang: %v", err)
	}
	if es.Name != "Espanol" {
		t.Fatalf("expected updated language name, got %s", es.Name)
	}

	del := clause.Association{Association: "Languages", Type: clause.OpDelete, Conditions: []clause.Expression{clause.Eq{Column: clause.Column{Name: "code"}, Value: "es"}}}
	if _, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(del).Update(ctx); err != nil {
		t.Fatalf("OpDelete m2m failed: %v", err)
	}
	AssertAssociationCount(t, user, "Languages", 1, "after delete one")
	// language row remains
	var count int64
	if err := DB.Model(&Language{}).Where("code = ?", "es").Count(&count).Error; err != nil {
		t.Fatalf("count lang: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected language row still exists, got %d", count)
	}
}

// Multi-owners: HasMany update and delete
func TestClauseAssociationSetUpdateAndDeleteManyOwnersHasMany(t *testing.T) {
	ctx := context.Background()
	u1 := User{Name: "MultiOwners-HasMany-1", Age: 21}
	u1.Pets = []*Pet{{Name: "p1"}}
	u2 := User{Name: "MultiOwners-HasMany-2", Age: 22}
	u2.Pets = []*Pet{{Name: "p2"}}
	if err := DB.Create(&u1).Error; err != nil {
		t.Fatalf("create u1: %v", err)
	}
	if err := DB.Create(&u2).Error; err != nil {
		t.Fatalf("create u2: %v", err)
	}
	AssertAssociationCount(t, u1, "Pets", 1, "before")
	AssertAssociationCount(t, u2, "Pets", 1, "before")

	upd := clause.Association{Association: "Pets", Type: clause.OpUpdate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "x"}}}
	if _, err := gorm.G[User](DB).Where("id IN ?", []uint{u1.ID, u2.ID}).Set(upd).Update(ctx); err != nil {
		t.Fatalf("OpUpdate has-many failed: %v", err)
	}
	var got1, got2 User
	if err := DB.Preload("Pets").First(&got1, u1.ID).Error; err != nil {
		t.Fatalf("load u1: %v", err)
	}
	if err := DB.Preload("Pets").First(&got2, u2.ID).Error; err != nil {
		t.Fatalf("load u2: %v", err)
	}
	if len(got1.Pets) != 1 || got1.Pets[0].Name != "x" {
		t.Fatalf("u1 pet not updated: %+v", got1.Pets)
	}
	if len(got2.Pets) != 1 || got2.Pets[0].Name != "x" {
		t.Fatalf("u2 pet not updated: %+v", got2.Pets)
	}

	del := clause.Association{Association: "Pets", Type: clause.OpDelete}
	if _, err := gorm.G[User](DB).Where("id IN ?", []uint{u1.ID, u2.ID}).Set(del).Update(ctx); err != nil {
		t.Fatalf("OpDelete has-many failed: %v", err)
	}
	AssertAssociationCount(t, u1, "Pets", 0, "after delete")
	AssertAssociationCount(t, u2, "Pets", 0, "after delete")
}

// Multi-owners: BelongsTo update and delete

func TestClauseAssociationSetUpdateAndDeleteManyOwnersBelongsTo(t *testing.T) {
	ctx := context.Background()

	// Create companies
	c1 := Company{Name: "Electronics"}
	c2 := Company{Name: "Books"}
	if err := DB.Create(&c1).Error; err != nil {
		t.Fatalf("create c1: %v", err)
	}
	if err := DB.Create(&c2).Error; err != nil {
		t.Fatalf("create c2: %v", err)
	}

	// Create users with companies
	u1 := User{Name: "John", Age: 25, CompanyID: &c1.ID}
	u2 := User{Name: "Jane", Age: 30, CompanyID: &c2.ID}
	if err := DB.Create(&u1).Error; err != nil {
		t.Fatalf("create u1: %v", err)
	}
	if err := DB.Create(&u2).Error; err != nil {
		t.Fatalf("create u2: %v", err)
	}

	// Verify associations exist
	AssertAssociationCount(t, &u1, "Company", 1, "before")
	AssertAssociationCount(t, &u2, "Company", 1, "before")

	// Update companies via OpUpdate for multiple users
	upd := clause.Association{Association: "Company", Type: clause.OpUpdate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "Category-New"}}}
	if _, err := gorm.G[User](DB).Where("id IN ?", []uint{u1.ID, u2.ID}).Set(upd).Update(ctx); err != nil {
		t.Fatalf("OpUpdate belongs-to failed: %v", err)
	}

	// Check if companies were updated
	var g1, g2 User
	if err := DB.Preload("Company").First(&g1, u1.ID).Error; err != nil {
		t.Fatalf("load u1: %v", err)
	}
	if err := DB.Preload("Company").First(&g2, u2.ID).Error; err != nil {
		t.Fatalf("load u2: %v", err)
	}

	if (g1.Company.ID == 0 || g1.Company.Name != "Category-New") || (g2.Company.ID == 0 || g2.Company.Name != "Category-New") {
		t.Fatalf("company names not updated: %+v, %+v", g1.Company, g2.Company)
	}

	// Unlink companies via OpUnlink for multiple users (instead of OpDelete which would try to delete the company records)
	unlink := clause.Association{Association: "Company", Type: clause.OpUnlink}
	if _, err := gorm.G[User](DB).Where("id IN ?", []uint{u1.ID, u2.ID}).Set(unlink).Update(ctx); err != nil {
		t.Fatalf("OpUnlink belongs-to failed: %v", err)
	}

	// Reload users to reflect the changes in the database
	if err := DB.First(&u1, u1.ID).Error; err != nil {
		t.Fatalf("reload u1: %v", err)
	}
	if err := DB.First(&u2, u2.ID).Error; err != nil {
		t.Fatalf("reload u2: %v", err)
	}

	// Check if company associations were cleared
	AssertAssociationCount(t, &u1, "Company", 0, "after unlink")
	AssertAssociationCount(t, &u2, "Company", 0, "after unlink")
}

// Multi-owners: Many2Many update and delete
func TestClauseAssociationSetUpdateAndDeleteManyOwnersMany2Many(t *testing.T) {
	ctx := context.Background()
	u1 := User{Name: "MultiOwners-M2M-1", Age: 21}
	u2 := User{Name: "MultiOwners-M2M-2", Age: 22}
	if err := DB.Create(&u1).Error; err != nil {
		t.Fatalf("create u1: %v", err)
	}
	if err := DB.Create(&u2).Error; err != nil {
		t.Fatalf("create u2: %v", err)
	}
	l1 := Language{Code: "zz", Name: "ZZ"}
	l2 := Language{Code: "yy", Name: "YY"}
	DB.FirstOrCreate(&l1, "code = ?", l1.Code)
	DB.FirstOrCreate(&l2, "code = ?", l2.Code)
	if err := DB.Model(&u1).Association("Languages").Append(&l1, &l2); err != nil {
		t.Fatalf("append u1: %v", err)
	}
	if err := DB.Model(&u2).Association("Languages").Append(&l1, &l2); err != nil {
		t.Fatalf("append u2: %v", err)
	}
	AssertAssociationCount(t, u1, "Languages", 2, "before")
	AssertAssociationCount(t, u2, "Languages", 2, "before")

	upd := clause.Association{Association: "Languages", Type: clause.OpUpdate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "ZZZ"}}, Conditions: []clause.Expression{clause.Eq{Column: clause.Column{Name: "code"}, Value: "zz"}}}
	if _, err := gorm.G[User](DB).Where("id IN ?", []uint{u1.ID, u2.ID}).Set(upd).Update(ctx); err != nil {
		t.Fatalf("OpUpdate m2m failed: %v", err)
	}
	var l Language
	if err := DB.First(&l, "code = ?", "zz").Error; err != nil {
		t.Fatalf("load lang: %v", err)
	}
	if l.Name != "ZZZ" {
		t.Fatalf("expected lang updated, got %s", l.Name)
	}

	del := clause.Association{Association: "Languages", Type: clause.OpDelete, Conditions: []clause.Expression{clause.Eq{Column: clause.Column{Name: "code"}, Value: "zz"}}}
	if _, err := gorm.G[User](DB).Where("id IN ?", []uint{u1.ID, u2.ID}).Set(del).Update(ctx); err != nil {
		t.Fatalf("OpDelete m2m failed: %v", err)
	}
	AssertAssociationCount(t, u1, "Languages", 1, "after delete")
	AssertAssociationCount(t, u2, "Languages", 1, "after delete")
}

// Test Set + Update with has-one (NamedPet) using OpCreate
func TestClauseAssociationSetUpdateHasOneCreateValues(t *testing.T) {
	ctx := context.Background()

	user := User{Name: "TestClauseAssociationSetUpdateHasOneCreateValues", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	assocOp := clause.Association{
		Association: "NamedPet",
		Type:        clause.OpCreate,
		Values:      []interface{}{Pet{Name: "named-pet"}},
	}

	rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx)
	if err != nil {
		t.Fatalf("Set Update has-one create values failed: %v", err)
	}
	if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
	}

	var updated User
	if err := DB.Preload("NamedPet").First(&updated, user.ID).Error; err != nil {
		t.Fatalf("failed to load user: %v", err)
	}
	if updated.NamedPet == nil || updated.NamedPet.Name != "named-pet" {
		t.Fatalf("expected named-pet created, got %+v", updated.NamedPet)
	}
}

// Test Set + Update to clear all has-many (Pets) via OpUnlink without conditions
func TestClauseAssociationSetUpdateHasManyClear(t *testing.T) {
	ctx := context.Background()

	user := User{Name: "TestClauseAssociationSetUpdateHasManyClear", Age: 25}
	user.Pets = []*Pet{{Name: "p1"}, {Name: "p2"}}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	AssertAssociationCount(t, user, "Pets", 2, "before clear")

	assocOp := clause.Association{Association: "Pets", Type: clause.OpUnlink}
	if rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx); err != nil {
		t.Fatalf("Set Update has-many clear failed: %v", err)
	} else if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
	}

	AssertAssociationCount(t, user, "Pets", 0, "after clear")
}

// Test Set + Update with many-to-many unlink specific records using conditions
func TestClauseAssociationSetUpdateManyToManyUnlink(t *testing.T) {
	ctx := context.Background()

	user := User{Name: "TestClauseAssociationSetUpdateManyToManyUnlink", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	langs := []Language{{Code: "es", Name: "Spanish"}, {Code: "de", Name: "German"}}
	for _, l := range langs {
		DB.FirstOrCreate(&l, "code = ?", l.Code)
	}

	// Associate both
	if err := DB.Model(&user).Association("Languages").Append(&langs); err != nil {
		t.Fatalf("failed to append languages: %v", err)
	}
	AssertAssociationCount(t, user, "Languages", 2, "before unlink")

	assocOp := clause.Association{
		Association: "Languages",
		Type:        clause.OpUnlink,
		Conditions:  []clause.Expression{clause.Eq{Column: clause.Column{Name: "code"}, Value: "es"}},
	}
	if rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx); err != nil {
		t.Fatalf("Set Update many-to-many unlink failed: %v", err)
	} else if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
	}
	AssertAssociationCount(t, user, "Languages", 1, "after unlink one")
}

// Test Set + Update with polymorphic has-many (Toys) using OpCreate
func TestClauseAssociationSetUpdatePolymorphicCreate(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "TestClauseAssociationSetUpdatePolymorphicCreate", Age: 25}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	assocOp := clause.Association{
		Association: "Toys",
		Type:        clause.OpCreate,
		Set:         []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "yo-yo"}},
	}
	if rows, err := gorm.G[User](DB).Where("id = ?", user.ID).Set(assocOp).Update(ctx); err != nil {
		t.Fatalf("Set Update polymorphic create failed: %v", err)
	} else if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
	}

	AssertAssociationCount(t, user, "Toys", 1, "after create toy")
}

// Test Set + Update across multiple owners
func TestClauseAssociationSetUpdateMultipleOwners(t *testing.T) {
	ctx := context.Background()

	u1 := User{Name: "SetUpdateMultipleOwners-1", Age: 20}
	u2 := User{Name: "SetUpdateMultipleOwners-2", Age: 21}
	if err := DB.Create(&u1).Error; err != nil {
		t.Fatalf("create u1: %v", err)
	}
	if err := DB.Create(&u2).Error; err != nil {
		t.Fatalf("create u2: %v", err)
	}

	assocOp := clause.Association{Association: "Pets", Type: clause.OpCreate, Set: []clause.Assignment{{Column: clause.Column{Name: "name"}, Value: "multi-pet"}}}
	if rows, err := gorm.G[User](DB).Where("name IN ?", []string{u1.Name, u2.Name}).Set(assocOp).Update(ctx); err != nil {
		t.Fatalf("Set Update multi owners failed: %v", err)
	} else if rows != 0 {
		t.Fatalf("expected 0 rows affected for association-only update, got %d", rows)
	}

	AssertAssociationCount(t, u1, "Pets", 1, "u1 after create")
	AssertAssociationCount(t, u2, "Pets", 1, "u2 after create")
}
