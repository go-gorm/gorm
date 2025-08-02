package tests_test

import (
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestDisableAssociationUpserts(t *testing.T) {
	// Setup test models
	type Profile struct {
		ID   uint
		Name string
	}

	type UserWithProfile struct {
		ID        uint
		Name      string
		ProfileID uint
		Profile   Profile
	}

	// Clean up and migrate
	DB.Migrator().DropTable(&UserWithProfile{}, &Profile{})
	if err := DB.AutoMigrate(&UserWithProfile{}, &Profile{}); err != nil {
		t.Fatalf("Failed to migrate tables: %v", err)
	}

	// Test 1: Default behavior (associations are created but not updated on conflict)
	t.Run("Default behavior", func(t *testing.T) {
		profile := Profile{ID: 1, Name: "Original Profile"}
		user := UserWithProfile{
			ID:        1,
			Name:      "Test User",
			ProfileID: 1,
			Profile:   profile,
		}

		// First create
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Verify profile was created
		var savedProfile Profile
		if err := DB.First(&savedProfile, 1).Error; err != nil {
			t.Fatalf("Failed to find created profile: %v", err)
		}
		if savedProfile.Name != "Original Profile" {
			t.Errorf("Expected profile name 'Original Profile', got '%s'", savedProfile.Name)
		}

		// Second create with updated profile (should not update existing profile by default)
		user.Profile.Name = "Updated Profile"
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user second time: %v", err)
		}

		// Verify profile was NOT updated (default DoNothing behavior)
		var unchangedProfile Profile
		if err := DB.First(&unchangedProfile, 1).Error; err != nil {
			t.Fatalf("Failed to find profile after second create: %v", err)
		}
		if unchangedProfile.Name != "Original Profile" {
			t.Errorf("Expected profile name to remain 'Original Profile', got '%s'", unchangedProfile.Name)
		}
	})

	// Test 2: With FullSaveAssociations (should update associations)
	t.Run("FullSaveAssociations behavior", func(t *testing.T) {
		// Clean up
		DB.Exec("DELETE FROM user_with_profiles")
		DB.Exec("DELETE FROM profiles")

		profile := Profile{ID: 2, Name: "Original Profile 2"}
		user := UserWithProfile{
			ID:        2,
			Name:      "Test User 2",
			ProfileID: 2,
			Profile:   profile,
		}

		// First create
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Second create with FullSaveAssociations (should update existing profile)
		user.Profile.Name = "Updated Profile 2"
		if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user with FullSaveAssociations: %v", err)
		}

		// Verify profile was updated
		var updatedProfile Profile
		if err := DB.First(&updatedProfile, 2).Error; err != nil {
			t.Fatalf("Failed to find profile after FullSaveAssociations create: %v", err)
		}
		if updatedProfile.Name != "Updated Profile 2" {
			t.Errorf("Expected profile name 'Updated Profile 2', got '%s'", updatedProfile.Name)
		}
	})

	// Test 3: With DisableAssociationUpserts (should never update associations)
	t.Run("DisableAssociationUpserts behavior", func(t *testing.T) {
		// Clean up
		DB.Exec("DELETE FROM user_with_profiles")
		DB.Exec("DELETE FROM profiles")

		profile := Profile{ID: 3, Name: "Original Profile 3"}
		user := UserWithProfile{
			ID:        3,
			Name:      "Test User 3",
			ProfileID: 3,
			Profile:   profile,
		}

		// Create with DisableAssociationUpserts enabled
		dbWithDisabledUpserts := DB.Session(&gorm.Session{DisableAssociationUpserts: true})

		// First create
		if err := dbWithDisabledUpserts.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Verify profile was created
		var savedProfile Profile
		if err := DB.First(&savedProfile, 3).Error; err != nil {
			t.Fatalf("Failed to find created profile: %v", err)
		}
		if savedProfile.Name != "Original Profile 3" {
			t.Errorf("Expected profile name 'Original Profile 3', got '%s'", savedProfile.Name)
		}

		// Second create with updated profile AND FullSaveAssociations
		// DisableAssociationUpserts should override FullSaveAssociations
		user.Profile.Name = "Should Not Update"
		if err := dbWithDisabledUpserts.Session(&gorm.Session{
			FullSaveAssociations:      true,
			DisableAssociationUpserts: true,
		}).Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user second time: %v", err)
		}

		// Verify profile was NOT updated despite FullSaveAssociations
		var unchangedProfile Profile
		if err := DB.First(&unchangedProfile, 3).Error; err != nil {
			t.Fatalf("Failed to find profile after second create: %v", err)
		}
		if unchangedProfile.Name != "Original Profile 3" {
			t.Errorf("Expected profile name to remain 'Original Profile 3' (DisableAssociationUpserts should override FullSaveAssociations), got '%s'", unchangedProfile.Name)
		}
	})

	// Test 4: Global DisableAssociationUpserts configuration
	t.Run("Global DisableAssociationUpserts configuration", func(t *testing.T) {
		// Clean up
		DB.Exec("DELETE FROM user_with_profiles")
		DB.Exec("DELETE FROM profiles")

		// Create a new DB instance with DisableAssociationUpserts enabled globally
		globalDB := DB.Session(&gorm.Session{DisableAssociationUpserts: true})

		profile := Profile{ID: 4, Name: "Original Profile 4"}
		user := UserWithProfile{
			ID:        4,
			Name:      "Test User 4",
			ProfileID: 4,
			Profile:   profile,
		}

		// First create
		if err := globalDB.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Second create with updated profile
		user.Profile.Name = "Should Not Update Global"
		if err := globalDB.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user second time: %v", err)
		}

		// Verify profile was NOT updated
		var unchangedProfile Profile
		if err := DB.First(&unchangedProfile, 4).Error; err != nil {
			t.Fatalf("Failed to find profile: %v", err)
		}
		if unchangedProfile.Name != "Original Profile 4" {
			t.Errorf("Expected profile name to remain 'Original Profile 4', got '%s'", unchangedProfile.Name)
		}
	})

	// Test 5: HasMany relationship
	t.Run("HasMany relationships", func(t *testing.T) {
		type Order struct {
			ID     uint
			Amount int
			UserID uint
		}

		type UserWithOrders struct {
			ID     uint
			Name   string
			Orders []Order
		}

		// Clean up and migrate
		DB.Migrator().DropTable(&UserWithOrders{}, &Order{})
		if err := DB.AutoMigrate(&UserWithOrders{}, &Order{}); err != nil {
			t.Fatalf("Failed to migrate tables: %v", err)
		}

		order := Order{ID: 1, Amount: 100}
		user := UserWithOrders{
			ID:     1,
			Name:   "User with Orders",
			Orders: []Order{order},
		}

		// First create
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user with orders: %v", err)
		}

		// Update order and create again with DisableAssociationUpserts
		user.Orders[0].Amount = 200
		if err := DB.Session(&gorm.Session{DisableAssociationUpserts: true}).Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user second time: %v", err)
		}

		// Verify order was NOT updated
		var unchangedOrder Order
		if err := DB.First(&unchangedOrder, 1).Error; err != nil {
			t.Fatalf("Failed to find order: %v", err)
		}
		if unchangedOrder.Amount != 100 {
			t.Errorf("Expected order amount to remain 100, got %d", unchangedOrder.Amount)
		}
	})
}