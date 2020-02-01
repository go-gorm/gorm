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
	Name                     string
	Type                     RelationshipType
	Field                    *Field
	Polymorphic              *Polymorphic
	References               []Reference
	Schema                   *Schema
	FieldSchema              *Schema
	JoinTable                *Schema
	ForeignKeys, PrimaryKeys []string
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
			Name:        field.Name,
			Field:       field,
			Schema:      schema,
			ForeignKeys: toColumns(field.TagSettings["FOREIGNKEY"]),
			PrimaryKeys: toColumns(field.TagSettings["PRIMARYKEY"]),
		}
	)

	if relation.FieldSchema, schema.err = Parse(fieldValue, schema.cacheStore, schema.namer); schema.err != nil {
		return
	}

	// Parse Polymorphic relations
	//
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
			schema.err = fmt.Errorf("invalid polymorphic type %v for %v on field %v, missing field %v", relation.FieldSchema, schema, field.Name, polymorphic+"Type")
		}

		if relation.Polymorphic.PolymorphicID == nil {
			schema.err = fmt.Errorf("invalid polymorphic type %v for %v on field %v, missing field %v", relation.FieldSchema, schema, field.Name, polymorphic+"ID")
		}

		if schema.err == nil {
			relation.References = append(relation.References, Reference{
				PriamryValue: relation.Polymorphic.Value,
				ForeignKey:   relation.Polymorphic.PolymorphicType,
			})

			primaryKeyField := schema.PrioritizedPrimaryField
			if len(relation.ForeignKeys) > 0 {
				if primaryKeyField = schema.LookUpField(relation.ForeignKeys[0]); primaryKeyField == nil || len(relation.ForeignKeys) > 1 {
					schema.err = fmt.Errorf("invalid polymorphic foreign keys %+v for %v on field %v", relation.ForeignKeys, schema, field.Name)
				}
			}
			relation.References = append(relation.References, Reference{
				PriamryKey:    primaryKeyField,
				ForeignKey:    relation.Polymorphic.PolymorphicType,
				OwnPriamryKey: true,
			})
		}

		relation.Type = "has"
	} else {
		switch field.FieldType.Kind() {
		case reflect.Struct:
			schema.guessRelation(relation, field, true)
		case reflect.Slice:
			schema.guessRelation(relation, field, true)
		default:
			schema.err = fmt.Errorf("unsupported data type %v for %v on field %v", relation.FieldSchema, schema, field.Name)
		}
	}

	if relation.Type == "has" {
		switch field.FieldType.Kind() {
		case reflect.Struct:
			relation.Type = HasOne
		case reflect.Slice:
			relation.Type = HasMany
		}
	}
}

func (schema *Schema) guessRelation(relation *Relationship, field *Field, guessHas bool) {
	var (
		primaryFields, foreignFields []*Field
		primarySchema, foreignSchema = schema, relation.FieldSchema
	)

	if !guessHas {
		primarySchema, foreignSchema = relation.FieldSchema, schema
	}

	reguessOrErr := func(err string, args ...interface{}) {
		if guessHas {
			schema.guessRelation(relation, field, false)
		} else {
			schema.err = fmt.Errorf(err, args...)
		}
	}

	if len(relation.ForeignKeys) > 0 {
		for _, foreignKey := range relation.ForeignKeys {
			if f := foreignSchema.LookUpField(foreignKey); f != nil {
				foreignFields = append(foreignFields, f)
			} else {
				reguessOrErr("unsupported relations %v for %v on field %v with foreign keys %v", relation.FieldSchema, schema, field.Name, relation.ForeignKeys)
				return
			}
		}
	} else {
		for _, primaryField := range primarySchema.PrimaryFields {
			if f := foreignSchema.LookUpField(field.Name + primaryField.Name); f != nil {
				foreignFields = append(foreignFields, f)
				primaryFields = append(primaryFields, primaryField)
			}
		}
	}

	if len(foreignFields) == 0 {
		reguessOrErr("failed to guess %v's relations with %v's field %v", relation.FieldSchema, schema, field.Name)
		return
	} else if len(relation.PrimaryKeys) > 0 {
		for idx, primaryKey := range relation.PrimaryKeys {
			if f := primarySchema.LookUpField(primaryKey); f != nil {
				if len(primaryFields) < idx+1 {
					primaryFields = append(primaryFields, f)
				} else if f != primaryFields[idx] {
					reguessOrErr("unsupported relations %v for %v on field %v with primary keys %v", relation.FieldSchema, schema, field.Name, relation.PrimaryKeys)
					return
				}
			} else {
				reguessOrErr("unsupported relations %v for %v on field %v with primary keys %v", relation.FieldSchema, schema, field.Name, relation.PrimaryKeys)
				return
			}
		}
	} else if len(primaryFields) == 0 {
		if len(foreignFields) == 1 {
			primaryFields = append(primaryFields, primarySchema.PrioritizedPrimaryField)
		} else if len(primarySchema.PrimaryFields) == len(foreignFields) {
			primaryFields = append(primaryFields, primarySchema.PrimaryFields...)
		} else {
			reguessOrErr("unsupported relations %v for %v on field %v", relation.FieldSchema, schema, field.Name)
			return
		}
	}

	// build references
	for idx, foreignField := range foreignFields {
		relation.References = append(relation.References, Reference{
			PriamryKey:    primaryFields[idx],
			ForeignKey:    foreignField,
			OwnPriamryKey: schema == primarySchema,
		})
	}

	if guessHas {
		relation.Type = "has"
	} else {
		relation.Type = "belongs_to"
	}
}

func (schema *Schema) parseMany2ManyRelation(relation *Relationship, field *Field) error {
	return nil
}
