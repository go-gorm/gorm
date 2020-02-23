package schema

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/jinzhu/inflection"
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
	foreignKeys, primaryKeys []string
}

type Polymorphic struct {
	PolymorphicID   *Field
	PolymorphicType *Field
	Value           string
}

type Reference struct {
	PrimaryKey    *Field
	PrimaryValue  string
	ForeignKey    *Field
	OwnPrimaryKey bool
}

func (schema *Schema) parseRelation(field *Field) {
	var (
		err        error
		fieldValue = reflect.New(field.IndirectFieldType).Interface()
		relation   = &Relationship{
			Name:        field.Name,
			Field:       field,
			Schema:      schema,
			foreignKeys: toColumns(field.TagSettings["FOREIGNKEY"]),
			primaryKeys: toColumns(field.TagSettings["REFERENCES"]),
		}
	)

	if relation.FieldSchema, _, err = Parse(fieldValue, schema.cacheStore, schema.namer); err != nil {
		schema.err = err
		return
	}

	if polymorphic, _ := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
		schema.buildPolymorphicRelation(relation, field, polymorphic)
	} else if many2many, _ := field.TagSettings["MANY2MANY"]; many2many != "" {
		schema.buildMany2ManyRelation(relation, field, many2many)
	} else {
		switch field.IndirectFieldType.Kind() {
		case reflect.Struct, reflect.Slice:
			schema.guessRelation(relation, field, true)
		default:
			schema.err = fmt.Errorf("unsupported data type %v for %v on field %v", relation.FieldSchema, schema, field.Name)
		}
	}

	if relation.Type == "has" {
		switch field.IndirectFieldType.Kind() {
		case reflect.Struct:
			relation.Type = HasOne
		case reflect.Slice:
			relation.Type = HasMany
		}
	}

	if schema.err == nil {
		schema.Relationships.Relations[relation.Name] = relation
		switch relation.Type {
		case HasOne:
			schema.Relationships.HasOne = append(schema.Relationships.HasOne, relation)
		case HasMany:
			schema.Relationships.HasMany = append(schema.Relationships.HasMany, relation)
		case BelongsTo:
			schema.Relationships.BelongsTo = append(schema.Relationships.BelongsTo, relation)
		case Many2Many:
			schema.Relationships.Many2Many = append(schema.Relationships.Many2Many, relation)
		}
	}
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
func (schema *Schema) buildPolymorphicRelation(relation *Relationship, field *Field, polymorphic string) {
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
			PrimaryValue: relation.Polymorphic.Value,
			ForeignKey:   relation.Polymorphic.PolymorphicType,
		})

		primaryKeyField := schema.PrioritizedPrimaryField
		if len(relation.foreignKeys) > 0 {
			if primaryKeyField = schema.LookUpField(relation.foreignKeys[0]); primaryKeyField == nil || len(relation.foreignKeys) > 1 {
				schema.err = fmt.Errorf("invalid polymorphic foreign keys %+v for %v on field %v", relation.foreignKeys, schema, field.Name)
			}
		}
		relation.References = append(relation.References, Reference{
			PrimaryKey:    primaryKeyField,
			ForeignKey:    relation.Polymorphic.PolymorphicID,
			OwnPrimaryKey: true,
		})
	}

	relation.Type = "has"
}

func (schema *Schema) buildMany2ManyRelation(relation *Relationship, field *Field, many2many string) {
	relation.Type = Many2Many

	var (
		err             error
		joinTableFields []reflect.StructField
		fieldsMap       = map[string]*Field{}
		ownFieldsMap    = map[string]bool{} // fix self join many2many
	)

	for _, s := range []*Schema{schema, relation.FieldSchema} {
		for _, primaryField := range s.PrimaryFields {
			fieldName := s.Name + primaryField.Name
			if _, ok := fieldsMap[fieldName]; ok {
				if field.Name != s.Name {
					fieldName = inflection.Singular(field.Name) + primaryField.Name
				} else {
					fieldName = s.Name + primaryField.Name + "Reference"
				}
			} else {
				ownFieldsMap[fieldName] = true
			}

			fieldsMap[fieldName] = primaryField
			joinTableFields = append(joinTableFields, reflect.StructField{
				Name:    fieldName,
				PkgPath: primaryField.StructField.PkgPath,
				Type:    primaryField.StructField.Type,
				Tag:     removeSettingFromTag(primaryField.StructField.Tag, "column"),
			})
		}
	}

	if relation.JoinTable, _, err = Parse(reflect.New(reflect.StructOf(joinTableFields)).Interface(), schema.cacheStore, schema.namer); err != nil {
		schema.err = err
	}
	relation.JoinTable.Name = many2many
	relation.JoinTable.Table = schema.namer.JoinTableName(many2many)

	// build references
	for _, f := range relation.JoinTable.Fields {
		relation.References = append(relation.References, Reference{
			PrimaryKey:    fieldsMap[f.Name],
			ForeignKey:    f,
			OwnPrimaryKey: schema == fieldsMap[f.Name].Schema && ownFieldsMap[f.Name],
		})
	}
	return
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

	if len(relation.foreignKeys) > 0 {
		for _, foreignKey := range relation.foreignKeys {
			if f := foreignSchema.LookUpField(foreignKey); f != nil {
				foreignFields = append(foreignFields, f)
			} else {
				reguessOrErr("unsupported relations %v for %v on field %v with foreign keys %v", relation.FieldSchema, schema, field.Name, relation.foreignKeys)
				return
			}
		}
	} else {
		for _, primaryField := range primarySchema.PrimaryFields {
			lookUpName := schema.Name + primaryField.Name
			if !guessHas {
				lookUpName = field.Name + primaryField.Name
			}

			if f := foreignSchema.LookUpField(lookUpName); f != nil {
				foreignFields = append(foreignFields, f)
				primaryFields = append(primaryFields, primaryField)
			}
		}
	}

	if len(foreignFields) == 0 {
		reguessOrErr("failed to guess %v's relations with %v's field %v 1 g %v", relation.FieldSchema, schema, field.Name, guessHas)
		return
	} else if len(relation.primaryKeys) > 0 {
		for idx, primaryKey := range relation.primaryKeys {
			if f := primarySchema.LookUpField(primaryKey); f != nil {
				if len(primaryFields) < idx+1 {
					primaryFields = append(primaryFields, f)
				} else if f != primaryFields[idx] {
					reguessOrErr("unsupported relations %v for %v on field %v with primary keys %v", relation.FieldSchema, schema, field.Name, relation.primaryKeys)
					return
				}
			} else {
				reguessOrErr("unsupported relations %v for %v on field %v with primary keys %v", relation.FieldSchema, schema, field.Name, relation.primaryKeys)
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
			PrimaryKey:    primaryFields[idx],
			ForeignKey:    foreignField,
			OwnPrimaryKey: schema == primarySchema && guessHas,
		})
	}

	if guessHas {
		relation.Type = "has"
	} else {
		relation.Type = BelongsTo
	}
}

type Constraint struct {
	Name            string
	Field           *Field
	Schema          *Schema
	ForeignKeys     []*Field
	ReferenceSchema *Schema
	References      []*Field
	OnDelete        string
	OnUpdate        string
}

func (rel *Relationship) ParseConstraint() *Constraint {
	str := rel.Field.TagSettings["CONSTRAINT"]
	if str == "-" {
		return nil
	}

	var (
		name     string
		idx      = strings.Index(str, ",")
		settings = ParseTagSetting(str, ",")
	)

	if idx != -1 && regexp.MustCompile("^[A-Za-z-_]+$").MatchString(str[0:idx]) {
		name = str[0:idx]
	} else {
		name = rel.Schema.namer.RelationshipFKName(*rel)
	}

	constraint := Constraint{
		Name:     name,
		Field:    rel.Field,
		OnUpdate: settings["ONUPDATE"],
		OnDelete: settings["ONDELETE"],
		Schema:   rel.Schema,
	}

	for _, ref := range rel.References {
		if ref.PrimaryKey != nil && !ref.OwnPrimaryKey {
			constraint.ForeignKeys = append(constraint.ForeignKeys, ref.ForeignKey)
			constraint.References = append(constraint.References, ref.PrimaryKey)
			constraint.ReferenceSchema = ref.PrimaryKey.Schema
		}
	}

	if rel.JoinTable != nil || constraint.ReferenceSchema == nil {
		return nil
	}

	return &constraint
}
