package clause

// AssociationOpType represents association operation types
type AssociationOpType int

const (
	OpUnlink AssociationOpType = iota // Unlink association
	OpDelete                          // Delete association records
	OpUpdate                          // Update association records
	OpCreate                          // Create association records with assignments
)

// Association represents an association operation
type Association struct {
	Association string            // Association name
	Type        AssociationOpType // Operation type
	Conditions  []Expression      // Filter conditions
	Set         []Assignment      // Assignment operations (for Update and Create)
	Values      []interface{}     // Values for Create operation
}

// AssociationAssigner is an interface for association operation providers
type AssociationAssigner interface {
	AssociationAssignments() []Association
}

// Assignments implements the Assigner interface so that AssociationOperation can be used as a Set method parameter
func (ao Association) Assignments() []Assignment {
	return []Assignment{}
}

// AssociationAssignments implements the AssociationAssigner interface
func (ao Association) AssociationAssignments() []Association {
	return []Association{ao}
}
