package clause

// AssociationOpType selects a generic Set association operation. The relationship kind
// determines whether foreign keys, related rows, or join rows are changed.
type AssociationOpType int

const (
	// OpUnlink removes association references by clearing foreign keys or deleting join
	// rows, leaving related entities where the relation permits.
	OpUnlink AssociationOpType = iota // Unlink association
	// OpDelete requests related-record deletion for non-join relations. Many-to-many
	// handling deletes join rows rather than related entities; soft-delete rules may apply.
	OpDelete // Delete association records
	// OpUpdate updates related-record fields using Association.Set and Conditions.
	OpUpdate // Update association records
	// OpCreate appends newly created or supplied association values to matching owners.
	OpCreate // Create association records with assignments
)

// Association describes a deferred generic Set operation on a named model relationship.
// It is an assignment provider, not SQL by itself. Operations can involve multiple
// statements; use an explicit transaction when owner and related changes must be
// atomic.
//
// Current many-to-many association unlink, delete, and update paths rebuild the owner
// EXISTS query without the chain's owner filters. They can therefore affect links or
// related rows belonging to other owners; Association.Conditions filters related
// records, not owners. Use explicit owner-key-qualified operations when owner scoping
// is required. An enclosing transaction provides atomicity but does not restore those
// filters.
type Association struct {
	// Association is the Go relationship field name resolved on the owner model.
	Association string // Association name
	// Type chooses unlink, delete, update, or create behavior.
	Type AssociationOpType // Operation type
	// Conditions filters related records for supported update/delete/unlink paths; it does
	// not by itself select owner records.
	Conditions []Expression // Filter conditions
	// Set supplies related-column assignments for update/create operations.
	Set []Assignment // Assignment operations (for Update and Create)
	// Values supplies records to append for create operations when Set is empty.
	Values []interface{} // Values for Create operation
}

// AssociationAssigner supplies relationship operations to generic Set. Such operations
// are separated from ordinary column assignments and executed first.
type AssociationAssigner interface {
	// AssociationAssignments returns deferred association operations in execution order.
	AssociationAssignments() []Association
}

// Assignments returns an empty ordinary-column list so Association satisfies Assigner.
// Generic Set recognizes AssociationAssigner to retrieve its actual operation.
func (ao Association) Assignments() []Assignment {
	return []Assignment{}
}

// AssociationAssignments returns this operation as a one-element slice.
func (ao Association) AssociationAssignments() []Association {
	return []Association{ao}
}
