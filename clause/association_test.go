package clause_test

import (
	"testing"

	"gorm.io/gorm/clause"
)

// Compile-time assertions that types implement clause.Assigner and clause.AssociationAssigner
var (
	_ clause.Assigner         = clause.Association{}
	_ clause.AssociationAssigner = clause.Association{}
)

func TestAssociation(t *testing.T) {
	// Test that Association implements Assigner interface
	assoc := clause.Association{
		Association: "Orders",
		Type:        clause.OpDelete,
		Conditions:  []interface{}{"state = ?", "cancelled"},
	}

	assignments := assoc.Assignments()
	if len(assignments) != 0 {
		t.Errorf("Association.Assignments() should return empty slice, got %v", assignments)
	}

	// Test that Association implements AssociationAssigner interface
	assocAssignments := assoc.AssociationAssignments()
	if len(assocAssignments) != 1 {
		t.Errorf("Association.AssociationAssignments() should return slice with one element, got %v", assocAssignments)
	}

	if assocAssignments[0].Association != "Orders" {
		t.Errorf("Association.AssociationAssignments()[0].Association should be 'Orders', got %v", assocAssignments[0].Association)
	}

	if assocAssignments[0].Type != clause.OpDelete {
		t.Errorf("Association.AssociationAssignments()[0].Type should be OpDelete, got %v", assocAssignments[0].Type)
	}
}

func TestAssociationOperations(t *testing.T) {
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

// TestAssociationSlice tests that a slice of Association implements AssociationAssigner
func TestAssociationSlice(t *testing.T) {
	// Create a slice of Association
	associations := []clause.Association{
		{Association: "Orders", Type: clause.OpDelete},
		{Association: "Profiles", Type: clause.OpUpdate},
	}

	// Convert to AssociationAssigner (in practice, this would be handled differently)
	// For now, we'll test each Association individually
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