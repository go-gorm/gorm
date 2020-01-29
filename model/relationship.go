package model

// RelationshipType relationship type
type RelationshipType string

const (
	HasOneRel    RelationshipType = "has_one"      // HasOneRel has one relationship
	HasManyRel   RelationshipType = "has_many"     // HasManyRel has many relationship
	BelongsToRel RelationshipType = "belongs_to"   // BelongsToRel belongs to relationship
	Many2ManyRel RelationshipType = "many_to_many" // Many2ManyRel many to many relationship
)

type Relationships struct {
	HasOne    map[string]*Relationship
	BelongsTo map[string]*Relationship
	HasMany   map[string]*Relationship
	Many2Many map[string]*Relationship
}

type Relationship struct {
	Type                   RelationshipType
	ForeignKeys            []*RelationField // self
	AssociationForeignKeys []*RelationField // association
	JoinTable              *JoinTable
}

type RelationField struct {
	*Field
	PolymorphicField *Field
	PolymorphicValue string
}

type JoinTable struct {
	Table                  string
	ForeignKeys            []*RelationField
	AssociationForeignKeys []*RelationField
}
