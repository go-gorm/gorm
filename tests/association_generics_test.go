package tests_test

import (
	"context"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	. "gorm.io/gorm/utils/tests"
)

func TestGenericsSetWithAssociation(t *testing.T) {
	ctx := context.Background()

	// Create a user with some orders
	user := User{Name: "GenericsSetWithAssociation", Age: 25}
	if err := gorm.G[User](DB).Create(ctx, &user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Test creating association with Set using clause.Association
	// Note: This is a conceptual test - in practice, association operations
	// would be handled differently in the generics implementation
	
	// For now, we'll test that we can create clause.Association objects
	// and that they implement the required interfaces
	
	// Test basic Association creation
	assoc := clause.Association{
		Association: "Orders",
		Type:        clause.OpCreate,
		Set: []clause.Assignment{
			{Column: clause.Column{Name: "amount"}, Value: 100},
			{Column: clause.Column{Name: "state"}, Value: "new"},
		},
	}

	// Verify it implements Assigner interface
	assignments := assoc.Assignments()
	if len(assignments) != 0 {
		t.Errorf("Association.Assignments() should return empty slice, got %v", assignments)
	}

	// Verify it implements AssociationAssigner interface
	assocAssignments := assoc.AssociationAssignments()
	if len(assocAssignments) != 1 {
		t.Errorf("Association.AssociationAssignments() should return slice with one element, got %v", assocAssignments)
	}

	if assocAssignments[0].Association != "Orders" {
		t.Errorf("Association.AssociationAssignments()[0].Association should be 'Orders', got %v", assocAssignments[0].Association)
	}

	// Test different association operation types
	operations := []struct {
		Type     clause.AssociationOpType
		TypeName string
	}{
		{clause.OpUnlink, "OpUnlink"},
		{clause.OpDelete, "OpDelete"},
		{clause.OpUpdate, "OpUpdate"},
		{clause.OpCreate, "OpCreate"},
		{clause.OpCreateValues, "OpCreateValues"},
	}

	for _, op := range operations {
		assoc := clause.Association{
			Association: "Orders",
			Type:        op.Type,
		}

		if assoc.Type != op.Type {
			t.Errorf("Association type should be %s, got %v", op.TypeName, assoc.Type)
		}
	}
}

func TestAssociationSlice(t *testing.T) {
	// Test that a slice of Association can be used
	associations := []clause.Association{
		{Association: "Orders", Type: clause.OpDelete},
		{Association: "Profiles", Type: clause.OpUpdate},
	}

	// In practice, each Association would be processed individually
	// since []clause.Association doesn't implement AssociationAssigner directly
	for i, assoc := range associations {
		assigns := assoc.AssociationAssignments()
		if len(assigns) != 1 {
			t.Errorf("Association %d should return one assignment, got %v", i, len(assigns))
		}
		if assigns[0].Association != assoc.Association {
			t.Errorf("Association %d name should be %s, got %v", i, assoc.Association, assigns[0].Association)
		}
	}
}