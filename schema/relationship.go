package schema

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/jinzhu/inflection"
	"gorm.io/gorm/clause"
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
	References               []*Reference
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

	cacheStore := schema.cacheStore
	if field.OwnerSchema != nil {
		cacheStore = field.OwnerSchema.cacheStore
	}

	if relation.FieldSchema, err = getOrParse(fieldValue, cacheStore, schema.namer); err != nil {
		schema.err = err
		return
	}

	if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
		schema.buildPolymorphicRelation(relation, field, polymorphic)
	} else if many2many := field.TagSettings["MANY2MANY"]; many2many != "" {
		schema.buildMany2ManyRelation(relation, field, many2many)
	} else {
		switch field.IndirectFieldType.Kind() {
		case reflect.Struct:
			schema.guessRelation(relation, field, guessBelongs)
		case reflect.Slice:
			schema.guessRelation(relation, field, guessHas)
		default:
			schema.err = fmt.Errorf("unsupported data type %v for %v on field %v", relation.FieldSchema, schema, field.Name)
		}
	}

	if relation.Type == "has" {
		if relation.FieldSchema != relation.Schema && relation.Polymorphic == nil {
			relation.FieldSchema.Relationships.Relations["_"+relation.Schema.Name+"_"+relation.Name] = relation
		}

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

	if value, ok := field.TagSettings["POLYMORPHICVALUE"]; ok {
		relation.Polymorphic.Value = strings.TrimSpace(value)
	}

	if relation.Polymorphic.PolymorphicType == nil {
		schema.err = fmt.Errorf("invalid polymorphic type %v for %v on field %v, missing field %v", relation.FieldSchema, schema, field.Name, polymorphic+"Type")
	}

	if relation.Polymorphic.PolymorphicID == nil {
		schema.err = fmt.Errorf("invalid polymorphic type %v for %v on field %v, missing field %v", relation.FieldSchema, schema, field.Name, polymorphic+"ID")
	}

	if schema.err == nil {
		relation.References = append(relation.References, &Reference{
			PrimaryValue: relation.Polymorphic.Value,
			ForeignKey:   relation.Polymorphic.PolymorphicType,
		})

		primaryKeyField := schema.PrioritizedPrimaryField
		if len(relation.foreignKeys) > 0 {
			if primaryKeyField = schema.LookUpField(relation.foreignKeys[0]); primaryKeyField == nil || len(relation.foreignKeys) > 1 {
				schema.err = fmt.Errorf("invalid polymorphic foreign keys %+v for %v on field %v", relation.foreignKeys, schema, field.Name)
			}
		}

		// use same data type for foreign keys
		relation.Polymorphic.PolymorphicID.DataType = primaryKeyField.DataType
		relation.Polymorphic.PolymorphicID.GORMDataType = primaryKeyField.GORMDataType
		if relation.Polymorphic.PolymorphicID.Size == 0 {
			relation.Polymorphic.PolymorphicID.Size = primaryKeyField.Size
		}

		relation.References = append(relation.References, &Reference{
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
		joinForeignKeys = toColumns(field.TagSettings["JOINFOREIGNKEY"])
		joinReferences  = toColumns(field.TagSettings["JOINREFERENCES"])
	)

	ownForeignFields := schema.PrimaryFields
	refForeignFields := relation.FieldSchema.PrimaryFields

	if len(relation.foreignKeys) > 0 {
		ownForeignFields = []*Field{}
		for _, foreignKey := range relation.foreignKeys {
			if field := schema.LookUpField(foreignKey); field != nil {
				ownForeignFields = append(ownForeignFields, field)
			} else {
				schema.err = fmt.Errorf("invalid foreign key: %v", foreignKey)
				return
			}
		}
	}

	if len(relation.primaryKeys) > 0 {
		refForeignFields = []*Field{}
		for _, foreignKey := range relation.primaryKeys {
			if field := relation.FieldSchema.LookUpField(foreignKey); field != nil {
				refForeignFields = append(refForeignFields, field)
			} else {
				schema.err = fmt.Errorf("invalid foreign key: %v", foreignKey)
				return
			}
		}
	}

	for idx, ownField := range ownForeignFields {
		joinFieldName := schema.Name + ownField.Name
		if len(joinForeignKeys) > idx {
			joinFieldName = strings.Title(joinForeignKeys[idx])
		}

		ownFieldsMap[joinFieldName] = true
		fieldsMap[joinFieldName] = ownField
		joinTableFields = append(joinTableFields, reflect.StructField{
			Name:    joinFieldName,
			PkgPath: ownField.StructField.PkgPath,
			Type:    ownField.StructField.Type,
			Tag:     removeSettingFromTag(ownField.StructField.Tag, "column", "autoincrement", "index", "unique", "uniqueindex"),
		})
	}

	for idx, relField := range refForeignFields {
		joinFieldName := relation.FieldSchema.Name + relField.Name
		if len(joinReferences) > idx {
			joinFieldName = strings.Title(joinReferences[idx])
		}

		if _, ok := ownFieldsMap[joinFieldName]; ok {
			if field.Name != relation.FieldSchema.Name {
				joinFieldName = inflection.Singular(field.Name) + relField.Name
			} else {
				joinFieldName += "Reference"
			}
		}

		fieldsMap[joinFieldName] = relField
		joinTableFields = append(joinTableFields, reflect.StructField{
			Name:    joinFieldName,
			PkgPath: relField.StructField.PkgPath,
			Type:    relField.StructField.Type,
			Tag:     removeSettingFromTag(relField.StructField.Tag, "column", "autoincrement", "index", "unique", "uniqueindex"),
		})
	}

	joinTableFields = append(joinTableFields, reflect.StructField{
		Name: schema.Name + field.Name,
		Type: schema.ModelType,
		Tag:  `gorm:"-"`,
	})

	if relation.JoinTable, err = Parse(reflect.New(reflect.StructOf(joinTableFields)).Interface(), schema.cacheStore, schema.namer); err != nil {
		schema.err = err
	}
	relation.JoinTable.Name = many2many
	relation.JoinTable.Table = schema.namer.JoinTableName(many2many)
	relation.JoinTable.PrimaryFields = make([]*Field, 0, len(relation.JoinTable.Fields))

	relName := relation.Schema.Name
	relRefName := relation.FieldSchema.Name
	if relName == relRefName {
		relRefName = relation.Field.Name
	}

	if _, ok := relation.JoinTable.Relationships.Relations[relName]; !ok {
		relation.JoinTable.Relationships.Relations[relName] = &Relationship{
			Name:        relName,
			Type:        BelongsTo,
			Schema:      relation.JoinTable,
			FieldSchema: relation.Schema,
		}
	} else {
		relation.JoinTable.Relationships.Relations[relName].References = []*Reference{}
	}

	if _, ok := relation.JoinTable.Relationships.Relations[relRefName]; !ok {
		relation.JoinTable.Relationships.Relations[relRefName] = &Relationship{
			Name:        relRefName,
			Type:        BelongsTo,
			Schema:      relation.JoinTable,
			FieldSchema: relation.FieldSchema,
		}
	} else {
		relation.JoinTable.Relationships.Relations[relRefName].References = []*Reference{}
	}

	// build references
	for _, f := range relation.JoinTable.Fields {
		if f.Creatable || f.Readable || f.Updatable {
			// use same data type for foreign keys
			f.DataType = fieldsMap[f.Name].DataType
			f.GORMDataType = fieldsMap[f.Name].GORMDataType
			if f.Size == 0 {
				f.Size = fieldsMap[f.Name].Size
			}
			relation.JoinTable.PrimaryFields = append(relation.JoinTable.PrimaryFields, f)
			ownPriamryField := schema == fieldsMap[f.Name].Schema && ownFieldsMap[f.Name]

			if ownPriamryField {
				joinRel := relation.JoinTable.Relationships.Relations[relName]
				joinRel.Field = relation.Field
				joinRel.References = append(joinRel.References, &Reference{
					PrimaryKey: fieldsMap[f.Name],
					ForeignKey: f,
				})
			} else {
				joinRefRel := relation.JoinTable.Relationships.Relations[relRefName]
				if joinRefRel.Field == nil {
					joinRefRel.Field = relation.Field
				}
				joinRefRel.References = append(joinRefRel.References, &Reference{
					PrimaryKey: fieldsMap[f.Name],
					ForeignKey: f,
				})
			}

			relation.References = append(relation.References, &Reference{
				PrimaryKey:    fieldsMap[f.Name],
				ForeignKey:    f,
				OwnPrimaryKey: ownPriamryField,
			})
		}
	}
}

type guessLevel int

const (
	guessBelongs guessLevel = iota
	guessEmbeddedBelongs
	guessHas
	guessEmbeddedHas
)

func (schema *Schema) guessRelation(relation *Relationship, field *Field, gl guessLevel) {
	var (
		primaryFields, foreignFields []*Field
		primarySchema, foreignSchema = schema, relation.FieldSchema
	)

	reguessOrErr := func() {
		switch gl {
		case guessBelongs:
			schema.guessRelation(relation, field, guessEmbeddedBelongs)
		case guessEmbeddedBelongs:
			schema.guessRelation(relation, field, guessHas)
		case guessHas:
			schema.guessRelation(relation, field, guessEmbeddedHas)
		// case guessEmbeddedHas:
		default:
			schema.err = fmt.Errorf("invalid field found for struct %v's field %v, need to define a valid foreign key for relations or it need to implement the Valuer/Scanner interface", schema, field.Name)
		}
	}

	switch gl {
	case guessBelongs:
		primarySchema, foreignSchema = relation.FieldSchema, schema
	case guessEmbeddedBelongs:
		if field.OwnerSchema != nil {
			primarySchema, foreignSchema = relation.FieldSchema, field.OwnerSchema
		} else {
			reguessOrErr()
			return
		}
	case guessHas:
	case guessEmbeddedHas:
		if field.OwnerSchema != nil {
			primarySchema, foreignSchema = field.OwnerSchema, relation.FieldSchema
		} else {
			reguessOrErr()
			return
		}
	}

	if len(relation.foreignKeys) > 0 {
		for _, foreignKey := range relation.foreignKeys {
			if f := foreignSchema.LookUpField(foreignKey); f != nil {
				foreignFields = append(foreignFields, f)
			} else {
				reguessOrErr()
				return
			}
		}
	} else {
		var primaryFields []*Field

		if len(relation.primaryKeys) > 0 {
			for _, primaryKey := range relation.primaryKeys {
				if f := primarySchema.LookUpField(primaryKey); f != nil {
					primaryFields = append(primaryFields, f)
				}
			}
		} else {
			primaryFields = primarySchema.PrimaryFields
		}

		for _, primaryField := range primaryFields {
			lookUpName := primarySchema.Name + primaryField.Name
			if gl == guessBelongs {
				lookUpName = field.Name + primaryField.Name
			}

			lookUpNames := []string{lookUpName}
			if len(primaryFields) == 1 {
				lookUpNames = append(lookUpNames, strings.TrimSuffix(lookUpName, primaryField.Name)+"ID")
				lookUpNames = append(lookUpNames, strings.TrimSuffix(lookUpName, primaryField.Name)+"Id")
				lookUpNames = append(lookUpNames, schema.namer.ColumnName(foreignSchema.Table, strings.TrimSuffix(lookUpName, primaryField.Name)+"ID"))
			}

			for _, name := range lookUpNames {
				if f := foreignSchema.LookUpField(name); f != nil {
					foreignFields = append(foreignFields, f)
					primaryFields = append(primaryFields, primaryField)
					break
				}
			}
		}
	}

	if len(foreignFields) == 0 {
		reguessOrErr()
		return
	} else if len(relation.primaryKeys) > 0 {
		for idx, primaryKey := range relation.primaryKeys {
			if f := primarySchema.LookUpField(primaryKey); f != nil {
				if len(primaryFields) < idx+1 {
					primaryFields = append(primaryFields, f)
				} else if f != primaryFields[idx] {
					reguessOrErr()
					return
				}
			} else {
				reguessOrErr()
				return
			}
		}
	} else if len(primaryFields) == 0 {
		if len(foreignFields) == 1 && primarySchema.PrioritizedPrimaryField != nil {
			primaryFields = append(primaryFields, primarySchema.PrioritizedPrimaryField)
		} else if len(primarySchema.PrimaryFields) == len(foreignFields) {
			primaryFields = append(primaryFields, primarySchema.PrimaryFields...)
		} else {
			reguessOrErr()
			return
		}
	}

	// build references
	for idx, foreignField := range foreignFields {
		// use same data type for foreign keys
		foreignField.DataType = primaryFields[idx].DataType
		foreignField.GORMDataType = primaryFields[idx].GORMDataType
		if foreignField.Size == 0 {
			foreignField.Size = primaryFields[idx].Size
		}

		relation.References = append(relation.References, &Reference{
			PrimaryKey:    primaryFields[idx],
			ForeignKey:    foreignField,
			OwnPrimaryKey: (schema == primarySchema && gl == guessHas) || (field.OwnerSchema == primarySchema && gl == guessEmbeddedHas),
		})
	}

	if gl == guessHas || gl == guessEmbeddedHas {
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
	}

	for _, ref := range rel.References {
		if ref.PrimaryKey != nil {
			constraint.ForeignKeys = append(constraint.ForeignKeys, ref.ForeignKey)
			constraint.References = append(constraint.References, ref.PrimaryKey)

			if ref.OwnPrimaryKey {
				constraint.Schema = ref.ForeignKey.Schema
				constraint.ReferenceSchema = rel.Schema
			} else {
				constraint.Schema = rel.Schema
				constraint.ReferenceSchema = ref.PrimaryKey.Schema
			}
		}
	}

	if rel.JoinTable != nil {
		return nil
	}

	return &constraint
}

func (rel *Relationship) ToQueryConditions(reflectValue reflect.Value) (conds []clause.Expression) {
	table := rel.FieldSchema.Table
	foreignFields := []*Field{}
	relForeignKeys := []string{}

	if rel.JoinTable != nil {
		table = rel.JoinTable.Table
		for _, ref := range rel.References {
			if ref.OwnPrimaryKey {
				foreignFields = append(foreignFields, ref.PrimaryKey)
				relForeignKeys = append(relForeignKeys, ref.ForeignKey.DBName)
			} else if ref.PrimaryValue != "" {
				conds = append(conds, clause.Eq{
					Column: clause.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
					Value:  ref.PrimaryValue,
				})
			} else {
				conds = append(conds, clause.Eq{
					Column: clause.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
					Value:  clause.Column{Table: rel.FieldSchema.Table, Name: ref.PrimaryKey.DBName},
				})
			}
		}
	} else {
		for _, ref := range rel.References {
			if ref.OwnPrimaryKey {
				relForeignKeys = append(relForeignKeys, ref.ForeignKey.DBName)
				foreignFields = append(foreignFields, ref.PrimaryKey)
			} else if ref.PrimaryValue != "" {
				conds = append(conds, clause.Eq{
					Column: clause.Column{Table: rel.FieldSchema.Table, Name: ref.ForeignKey.DBName},
					Value:  ref.PrimaryValue,
				})
			} else {
				relForeignKeys = append(relForeignKeys, ref.PrimaryKey.DBName)
				foreignFields = append(foreignFields, ref.ForeignKey)
			}
		}
	}

	_, foreignValues := GetIdentityFieldValuesMap(reflectValue, foreignFields)
	column, values := ToQueryValues(table, relForeignKeys, foreignValues)

	conds = append(conds, clause.IN{Column: column, Values: values})
	return
}
