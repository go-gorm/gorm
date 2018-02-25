package schema

import "testing"

type HasOne struct {
	ID         int
	MyStructID uint
}

type HasMany struct {
	ID         int
	MyStructID uint
	Name       string
}

type Many2Many struct {
	ID   int
	Name string
}

func TestBelongsToRel(t *testing.T) {
	type BelongsTo struct {
		ID   int
		Name string
	}

	type MyStruct struct {
		ID          int
		Name        string
		BelongsToID uint
		BelongsTo   BelongsTo
	}

	schema := Parse(&MyStruct{})
	compareFields(schema.Fields, []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"Name"}, IsNormal: true},
		{DBName: "belongs_to_id", Name: "BelongsToID", BindNames: []string{"BelongsToID"}, IsNormal: true, IsForeignKey: true},
		{DBName: "belongs_to", Name: "BelongsTo", BindNames: []string{"BelongsTo"}, Relationship: &Relationship{Kind: "belongs_to", ForeignKey: []string{"belongs_to_id"}, AssociationForeignKey: []string{"id"}}},
	}, t)

	type MyStruct2 struct {
		ID           int `gorm:"column:my_id"`
		Name         string
		BelongsToKey uint
		BelongsTo    BelongsTo `gorm:"foreignkey:BelongsToKey"`
	}

	schema2 := Parse(&MyStruct2{})
	compareFields(schema2.Fields, []*Field{
		{DBName: "my_id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true, TagSettings: map[string]string{"COLUMN": "my_id"}},
		{DBName: "name", Name: "Name", BindNames: []string{"Name"}, IsNormal: true},
		{DBName: "belongs_to_key", Name: "BelongsToKey", BindNames: []string{"BelongsToKey"}, IsNormal: true, IsForeignKey: true},
		{DBName: "belongs_to", Name: "BelongsTo", BindNames: []string{"BelongsTo"}, Relationship: &Relationship{Kind: "belongs_to", ForeignKey: []string{"belongs_to_key"}, AssociationForeignKey: []string{"id"}}, TagSettings: map[string]string{"FOREIGNKEY": "BelongsToKey"}},
	}, t)

	type BelongsTo3 struct {
		ID   int `gorm:"column:my_id"`
		Name string
	}

	type MyStruct3 struct {
		ID           int
		Name         string
		BelongsToKey uint
		BelongsTo    BelongsTo3 `gorm:"foreignkey:BelongsToKey"`
	}

	schema3 := Parse(&MyStruct3{})
	compareFields(schema3.Fields, []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"Name"}, IsNormal: true},
		{DBName: "belongs_to_key", Name: "BelongsToKey", BindNames: []string{"BelongsToKey"}, IsNormal: true, IsForeignKey: true},
		{DBName: "belongs_to", Name: "BelongsTo", BindNames: []string{"BelongsTo"}, Relationship: &Relationship{Kind: "belongs_to", ForeignKey: []string{"belongs_to_key"}, AssociationForeignKey: []string{"my_id"}}, TagSettings: map[string]string{"FOREIGNKEY": "BelongsToKey"}},
	}, t)
}

func TestSelfReferenceBelongsToRel(t *testing.T) {
	type MyStruct struct {
		ID          int
		Name        string
		BelongsToID int
		BelongsTo   *MyStruct
	}

	// user1 belongs to user2, when creating, will create user2 first
	schema := Parse(&MyStruct{})
	compareFields(schema.Fields, []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"Name"}, IsNormal: true},
		{DBName: "belongs_to_id", Name: "BelongsToID", BindNames: []string{"BelongsToID"}, IsNormal: true, IsForeignKey: true},
		{DBName: "belongs_to", Name: "BelongsTo", BindNames: []string{"BelongsTo"}, Relationship: &Relationship{Kind: "belongs_to", ForeignKey: []string{"belongs_to_id"}, AssociationForeignKey: []string{"id"}}},
	}, t)

	type MyStruct2 struct {
		ID           int
		Name         string
		BelongsToKey int
		BelongsTo    *MyStruct2 `gorm:"rel:belongs_to;foreignkey:BelongsToKey"`
	}

	// user1 belongs to user2, when creating, will create user2 first
	schema2 := Parse(&MyStruct2{})
	compareFields(schema2.Fields, []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"Name"}, IsNormal: true},
		{DBName: "belongs_to_key", Name: "BelongsToKey", BindNames: []string{"BelongsToKey"}, IsNormal: true, IsForeignKey: true},
		{DBName: "belongs_to", Name: "BelongsTo", BindNames: []string{"BelongsTo"}, Relationship: &Relationship{Kind: "belongs_to", ForeignKey: []string{"belongs_to_key"}, AssociationForeignKey: []string{"id"}}, TagSettings: map[string]string{"FOREIGNKEY": "BelongsToKey"}},
	}, t)
}

func TestHasOneRel(t *testing.T) {
	type MyStruct struct {
		ID     int
		Name   string
		HasOne HasOne
	}

	Parse(&MyStruct{})
}

func TestSelfReferenceHasOneRel(t *testing.T) {
	type MyStruct struct {
		ID          int
		Name        string
		BelongsToID int
		BelongsTo   *MyStruct
	}

	// user1 belongs to user2, when creating, will create user2 first
	schema := Parse(&MyStruct{})
	compareFields(schema.Fields, []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"Name"}, IsNormal: true},
		{DBName: "belongs_to_id", Name: "BelongsToID", BindNames: []string{"BelongsToID"}, IsNormal: true, IsForeignKey: true},
		{DBName: "belongs_to", Name: "BelongsTo", BindNames: []string{"BelongsTo"}, Relationship: &Relationship{Kind: "belongs_to", ForeignKey: []string{"belongs_to_id"}, AssociationForeignKey: []string{"id"}}},
	}, t)
}

func TestPolymorphicHasOneRel(t *testing.T) {
	type HasOne struct {
		ID        int
		Name      string
		OwnerType string
		OwnerID   string
	}

	type MyStruct struct {
		ID     int
		Name   string
		HasOne HasOne `gorm:"polymorphic:Owner"`
	}

	schema := Parse(&MyStruct{})
	compareFields(schema.Fields, []*Field{
		{DBName: "id", Name: "ID", BindNames: []string{"ID"}, IsNormal: true, IsPrimaryKey: true},
		{DBName: "name", Name: "Name", BindNames: []string{"Name"}, IsNormal: true},
		{DBName: "has_one", Name: "HasOne", BindNames: []string{"HasOne"}, Relationship: &Relationship{Kind: "has_one", PolymorphicType: "OwnerType", PolymorphicDBName: "owner_type", PolymorphicValue: "my_struct", ForeignKey: []string{"owner_id"}, AssociationForeignKey: []string{"id"}}, TagSettings: map[string]string{"POLYMORPHIC": "Owner"}},
	}, t)
}

func TestHasManyRel(t *testing.T) {
	type MyStruct struct {
		ID      int
		Name    string
		HasMany []HasMany
	}

	Parse(&MyStruct{})
}

func TestManyToManyRel(t *testing.T) {
	type MyStruct struct {
		ID      int
		Name    string
		HasMany []HasMany
	}

	Parse(&MyStruct{})
}
