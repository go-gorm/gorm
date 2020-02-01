package schema

import (
	"fmt"
	"reflect"
	"strings"
)

// RelationshipType relationship type
type RelationshipType string

const (
	HasOne    RelationshipType = "has_one"      // HasOneRel has one relationship
	HasMany   RelationshipType = "has_many"     // HasManyRel has many relationship
	BelongsTo RelationshipType = "belongs_to"   // BelongsToRel belongs to relationship
	Many2Many RelationshipType = "many_to_many" // Many2ManyRel many to many relationship
)

type Relationships struct {
	HasOne    []*Relationship
	BelongsTo []*Relationship
	HasMany   []*Relationship
	Many2Many []*Relationship
	Relations map[string]*Relationship
}

type Relationship struct {
	Name                                string
	Type                                RelationshipType
	Field                               *Field
	Polymorphic                         *Polymorphic
	References                          []Reference
	Schema                              *Schema
	FieldSchema                         *Schema
	JoinTable                           *Schema
	ForeignKeys, AssociationForeignKeys []string
}

type Polymorphic struct {
	PolymorphicID   *Field
	PolymorphicType *Field
	Value           string
}

type Reference struct {
	PriamryKey    *Field
	PriamryValue  string
	ForeignKey    *Field
	OwnPriamryKey bool
}

func (schema *Schema) parseRelation(field *Field) {
	var (
		fieldValue = reflect.New(field.FieldType).Interface()
		relation   = &Relationship{
			Name:                   field.Name,
			Field:                  field,
			Schema:                 schema,
			Type:                   RelationshipType(strings.ToLower(strings.TrimSpace(field.TagSettings["REL"]))),
			ForeignKeys:            toColumns(field.TagSettings["FOREIGNKEY"]),
			AssociationForeignKeys: toColumns(field.TagSettings["ASSOCIATION_FOREIGNKEY"]),
		}
	)

	if relation.FieldSchema, schema.err = Parse(fieldValue, schema.cacheStore, schema.namer); schema.err != nil {
		return
	}

	// User has many Toys, its `Polymorphic` is `Owner`, Pet has one Toy, its `Polymorphic` is `Owner`
	//     type User struct {
	//       Toys []Toy `gorm:"polymorphic:Owner;"`
	//     }
	//     type Pet struct {
	//       Toy Toy `gorm:"polymorphic:Owner;"`
	//     }
	//     type Toy struct {
	//       OwnerID   int
	//       OwnerType string
	//     }
	if polymorphic, _ := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
		relation.Polymorphic = &Polymorphic{
			Value:           schema.Table,
			PolymorphicType: relation.FieldSchema.FieldsByName[polymorphic+"Type"],
			PolymorphicID:   relation.FieldSchema.FieldsByName[polymorphic+"ID"],
		}

		if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
			relation.Polymorphic.Value = strings.TrimSpace(value)
		}

		if relation.Polymorphic.PolymorphicType == nil {
			schema.err = fmt.Errorf("invalid polymorphic type: %v for %v on field %v, missing field %v", relation.FieldSchema, schema, field.Name, polymorphic+"Type")
		}

		if relation.Polymorphic.PolymorphicID == nil {
			schema.err = fmt.Errorf("invalid polymorphic type: %v for %v on field %v, missing field %v", relation.FieldSchema, schema, field.Name, polymorphic+"ID")
		}

		if schema.err == nil {
			relation.References = append(relation.References, Reference{
				PriamryValue: relation.Polymorphic.Value,
				ForeignKey:   relation.Polymorphic.PolymorphicType,
			})

			primaryKeyField := schema.PrioritizedPrimaryField
			if len(relation.ForeignKeys) > 0 {
				if primaryKeyField = schema.LookUpField(relation.ForeignKeys[0]); primaryKeyField == nil || len(relation.ForeignKeys) > 1 {
					schema.err = fmt.Errorf("invalid polymorphic foreign key: %+v for %v on field %v", relation.ForeignKeys, schema, field.Name)
				}
			}
			relation.References = append(relation.References, Reference{
				PriamryKey:    primaryKeyField,
				ForeignKey:    relation.Polymorphic.PolymorphicType,
				OwnPriamryKey: true,
			})
		}

		switch field.FieldType.Kind() {
		case reflect.Struct:
			relation.Type = HasOne
		case reflect.Slice:
			relation.Type = HasMany
		}
		return
	}

	switch field.FieldType.Kind() {
	case reflect.Struct:
		schema.parseStructRelation(relation, field)
	case reflect.Slice:
		schema.parseSliceRelation(relation, field)
	default:
		schema.err = fmt.Errorf("unsupported data type: %v (in %v#%v ", field.FieldType.PkgPath(), schema, field.Name)
	}
}

func (schema *Schema) parseStructRelation(relation *Relationship, field *Field) error {
	return nil
}

func (schema *Schema) parseSliceRelation(relation *Relationship, field *Field) error {
	return nil
}
