package schema

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/jinzhu/inflection"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"gorm.io/gorm/clause"
)

// RelationshipType classifies where an association stores its foreign keys and whether
// it uses a join table.
type RelationshipType string

const (
	// HasOne identifies a single associated model whose foreign key refers to the owner.
	HasOne RelationshipType = "has_one" // HasOneRel has one relationship
	// HasMany identifies a collection of associated models whose foreign keys refer to the
	// owner.
	HasMany RelationshipType = "has_many" // HasManyRel has many relationship
	// BelongsTo identifies an association whose foreign key is stored on the owner and
	// refers to the associated model.
	BelongsTo RelationshipType = "belongs_to" // BelongsToRel belongs to relationship
	// Many2Many identifies an association linked through a join table containing keys for
	// both models.
	Many2Many RelationshipType = "many_to_many" // Many2ManyRel many to many relationship
	// has is the intermediate owner-to-child relationship kind, resolved to HasOne or
	// HasMany from the field shape.
	has RelationshipType = "has"
)

// Relationships groups a schema's associations by kind, by field name, and by embedded
// path. It is built during schema parsing and shared by cached schemas; callers should
// treat it as metadata rather than modifying its maps during queries.
type Relationships struct {
	// HasOne lists single-record associations whose foreign keys are on the associated
	// model.
	HasOne []*Relationship
	// BelongsTo lists associations whose foreign keys are on this model.
	BelongsTo []*Relationship
	// HasMany lists collection associations whose foreign keys are on the associated model.
	HasMany []*Relationship
	// Many2Many lists associations that use join tables.
	Many2Many []*Relationship
	// Relations indexes associations by field name, and can include internal reverse
	// entries named _Owner_Relation. Explicit top-level fields take precedence over
	// embedded-name collisions.
	Relations map[string]*Relationship

	// EmbeddedRelations forms a tree for explicitly embedded association paths, used by
	// nested preloading.
	EmbeddedRelations map[string]*Relationships

	// Mux protects relationship updates performed during recursive schema parsing. It does
	// not make arbitrary Schema mutation or unsynchronized map access safe.
	Mux sync.RWMutex
}

// Relationship describes the association declared by a model field. References explains
// key direction; Schema is the owner, FieldSchema is the associated model, and
// JoinTable is populated for many-to-many relationships. It describes mapping, not
// whether rows have been loaded.
type Relationship struct {
	// Name is the Go association field name used by Preload, Joins, and Association.
	Name string
	// Type is the resolved cardinality and key direction of the association.
	Type RelationshipType
	// Field is the model field declaring the association.
	Field *Field
	// Polymorphic records discriminator and owner-ID fields when different owner types
	// share an associated table; otherwise it is nil.
	Polymorphic *Polymorphic
	// References contains key pairs and optional fixed discriminator values. Composite
	// relationships preserve key-pair order.
	References []*Reference
	// Schema is the model declaring the association.
	Schema *Schema
	// FieldSchema is the schema of the associated field's model.
	FieldSchema *Schema
	// JoinTable describes the intermediate table for a many-to-many association; it is nil
	// for direct associations.
	JoinTable *Schema
	// foreignKeys and primaryKeys retain parsed foreignKey and references tag names for
	// relationship inference. Their owning schema depends on the association direction.
	foreignKeys, primaryKeys []string
}

// Polymorphic identifies an associated table's owner ID and discriminator columns. The
// discriminator defaults to the owner table name and can be overridden with
// polymorphicValue.
type Polymorphic struct {
	// PolymorphicID is the associated model field storing the owner key.
	PolymorphicID *Field
	// PolymorphicType is the associated model field storing the owner-type discriminator.
	PolymorphicType *Field
	// Value is the fixed discriminator written and matched for this owner type.
	Value string
}

// Reference links a primary key to a foreign key or binds a foreign key to a fixed
// polymorphic discriminator. Use OwnPrimaryKey to determine whether the key comes from
// the association owner.
type Reference struct {
	// PrimaryKey is the referenced key field. It can be nil when PrimaryValue supplies a
	// fixed discriminator instead.
	PrimaryKey *Field
	// PrimaryValue is a fixed discriminator value used instead of reading a primary-key
	// field when nonempty.
	PrimaryValue string
	// ForeignKey is the field storing the key or discriminator, on the owner, associated
	// model, or join table according to the relationship.
	ForeignKey *Field
	// OwnPrimaryKey means PrimaryKey belongs to the association owner. False normally means
	// the associated model provides the primary key; inspect PrimaryValue for fixed-value
	// references.
	OwnPrimaryKey bool
}

// parseRelation parses an association field, choosing polymorphic and many-to-many tags
// before ordinary key inference. It resolves the associated schema through the
// recursion-safe cache, registers successful relations and reverse metadata, and
// records failures in schema.err.
func (schema *Schema) parseRelation(field *Field) *Relationship {
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

	if relation.FieldSchema, err = getOrParse(fieldValue, schema.cacheStore, schema.namer); err != nil {
		schema.err = fmt.Errorf("failed to parse field: %s, error: %w", field.Name, err)
		return nil
	}

	if hasPolymorphicRelation(field.TagSettings) {
		schema.buildPolymorphicRelation(relation, field)
	} else if many2many := field.TagSettings["MANY2MANY"]; many2many != "" {
		schema.buildMany2ManyRelation(relation, field, many2many)
	} else if belongsTo := field.TagSettings["BELONGSTO"]; belongsTo != "" {
		schema.guessRelation(relation, field, guessBelongs)
	} else {
		switch field.IndirectFieldType.Kind() {
		case reflect.Struct:
			schema.guessRelation(relation, field, guessGuess)
		case reflect.Slice:
			schema.guessRelation(relation, field, guessHas)
		default:
			schema.err = fmt.Errorf("unsupported data type %v for %v on field %s", relation.FieldSchema, schema,
				field.Name)
		}
	}

	if relation.Type == has {
		if relation.FieldSchema != relation.Schema && relation.Polymorphic == nil && field.OwnerSchema == nil {
			relation.FieldSchema.Relationships.Mux.Lock()
			relation.FieldSchema.Relationships.Relations["_"+relation.Schema.Name+"_"+relation.Name] = relation
			relation.FieldSchema.Relationships.Mux.Unlock()
		}

		switch field.IndirectFieldType.Kind() {
		case reflect.Struct:
			relation.Type = HasOne
		case reflect.Slice:
			relation.Type = HasMany
		}
	}

	if schema.err == nil {
		schema.setRelation(relation)
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

	return relation
}

// hasPolymorphicRelation reports whether the tags contain polymorphic, or both
// polymorphicType and polymorphicId. Presence selects polymorphic parsing; the target
// fields are validated separately.
func hasPolymorphicRelation(tagSettings map[string]string) bool {
	if _, ok := tagSettings["POLYMORPHIC"]; ok {
		return true
	}

	_, hasType := tagSettings["POLYMORPHICTYPE"]
	_, hasId := tagSettings["POLYMORPHICID"]

	return hasType && hasId
}

// setRelation registers the named association under the relationship mutex and builds
// its explicit embedded-path tree. Existing non-embedded names retain precedence over
// collisions.
func (schema *Schema) setRelation(relation *Relationship) {
	schema.Relationships.Mux.Lock()
	defer schema.Relationships.Mux.Unlock()

	// set non-embedded relation
	if rel := schema.Relationships.Relations[relation.Name]; rel != nil {
		if len(rel.Field.BindNames) > 1 {
			schema.Relationships.Relations[relation.Name] = relation
		}
	} else {
		schema.Relationships.Relations[relation.Name] = relation
	}

	// set embedded relation
	if len(relation.Field.EmbeddedBindNames) <= 1 {
		return
	}
	relationships := &schema.Relationships
	for i, name := range relation.Field.EmbeddedBindNames {
		if i < len(relation.Field.EmbeddedBindNames)-1 {
			if relationships.EmbeddedRelations == nil {
				relationships.EmbeddedRelations = map[string]*Relationships{}
			}
			if r := relationships.EmbeddedRelations[name]; r == nil {
				relationships.EmbeddedRelations[name] = &Relationships{}
			}
			relationships = relationships.EmbeddedRelations[name]
		} else {
			if relationships.Relations == nil {
				relationships.Relations = map[string]*Relationship{}
			}
			relationships.Relations[relation.Name] = relation
		}
	}
}

// buildPolymorphicRelation resolves the associated model's discriminator and owner-ID
// fields from polymorphic, polymorphicType, and polymorphicId tags. It defaults the
// discriminator to the owner table, supports polymorphicValue, copies compatible key
// type metadata, and records invalid fields or keys in schema.err.
//
// For Toys []Toy tagged gorm:"polymorphic:Owner", Toy normally supplies OwnerID and
// OwnerType. The relationship contains one fixed OwnerType reference and one
// owner-primary-key reference; parseRelation resolves its final HasOne or HasMany kind.
func (schema *Schema) buildPolymorphicRelation(relation *Relationship, field *Field) {
	polymorphic := field.TagSettings["POLYMORPHIC"]

	relation.Polymorphic = &Polymorphic{
		Value: schema.Table,
	}

	var (
		typeName = polymorphic + "Type"
		typeId   = polymorphic + "ID"
	)

	if value, ok := field.TagSettings["POLYMORPHICTYPE"]; ok {
		typeName = strings.TrimSpace(value)
	}

	if value, ok := field.TagSettings["POLYMORPHICID"]; ok {
		typeId = strings.TrimSpace(value)
	}

	relation.Polymorphic.PolymorphicType = relation.FieldSchema.FieldsByName[typeName]
	relation.Polymorphic.PolymorphicID = relation.FieldSchema.FieldsByName[typeId]

	if value, ok := field.TagSettings["POLYMORPHICVALUE"]; ok {
		relation.Polymorphic.Value = strings.TrimSpace(value)
	}

	if relation.Polymorphic.PolymorphicType == nil {
		schema.err = fmt.Errorf("invalid polymorphic type %v for %v on field %s, missing field %s",
			relation.FieldSchema, schema, field.Name, polymorphic+"Type")
	}

	if relation.Polymorphic.PolymorphicID == nil {
		schema.err = fmt.Errorf("invalid polymorphic type %v for %v on field %s, missing field %s",
			relation.FieldSchema, schema, field.Name, polymorphic+"ID")
	}

	if schema.err == nil {
		relation.References = append(relation.References, &Reference{
			PrimaryValue: relation.Polymorphic.Value,
			ForeignKey:   relation.Polymorphic.PolymorphicType,
		})

		primaryKeyField := schema.PrioritizedPrimaryField
		if len(relation.foreignKeys) > 0 {
			if primaryKeyField = schema.LookUpField(relation.foreignKeys[0]); primaryKeyField == nil || len(relation.foreignKeys) > 1 {
				schema.err = fmt.Errorf("invalid polymorphic foreign keys %+v for %v on field %s", relation.foreignKeys,
					schema, field.Name)
			}
		}

		if primaryKeyField == nil {
			schema.err = fmt.Errorf("invalid polymorphic type %v for %v on field %s, missing primaryKey field",
				relation.FieldSchema, schema, field.Name)
			return
		}

		// use same data type for foreign keys
		if copyableDataType(primaryKeyField.DataType) {
			relation.Polymorphic.PolymorphicID.DataType = primaryKeyField.DataType
		}
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

	relation.Type = has
}

// buildMany2ManyRelation constructs the join-table schema and key references for a
// many2many tag. Owner and related primary keys are defaults; foreignKey/references
// select source fields and joinForeignKey/joinReferences select join fields. It handles
// self-association name collisions, removes inappropriate inherited
// column/index/auto-increment tags, and marks join keys as primary fields.
//
// This builds metadata only. Table creation and custom join-model configuration are
// handled by the migrator and DB.SetupJoinTable respectively.
func (schema *Schema) buildMany2ManyRelation(relation *Relationship, field *Field, many2many string) {
	relation.Type = Many2Many

	var (
		err             error
		joinTableFields []reflect.StructField
		fieldsMap       = map[string]*Field{}
		ownFieldsMap    = map[string]*Field{} // fix self join many2many
		referFieldsMap  = map[string]*Field{}
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
				schema.err = fmt.Errorf("invalid foreign key: %s", foreignKey)
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
				schema.err = fmt.Errorf("invalid foreign key: %s", foreignKey)
				return
			}
		}
	}

	for idx, ownField := range ownForeignFields {
		joinFieldName := cases.Title(language.Und, cases.NoLower).String(schema.Name) + ownField.Name
		if len(joinForeignKeys) > idx {
			joinFieldName = cases.Title(language.Und, cases.NoLower).String(joinForeignKeys[idx])
		}

		ownFieldsMap[joinFieldName] = ownField
		fieldsMap[joinFieldName] = ownField
		joinTableFields = append(joinTableFields, reflect.StructField{
			Name:    joinFieldName,
			PkgPath: ownField.StructField.PkgPath,
			Type:    ownField.StructField.Type,
			Tag: removeSettingFromTag(appendSettingFromTag(ownField.StructField.Tag, "primaryKey"),
				"column", "autoincrement", "index", "unique", "uniqueindex"),
		})
	}

	for idx, relField := range refForeignFields {
		joinFieldName := cases.Title(language.Und, cases.NoLower).String(relation.FieldSchema.Name) + relField.Name

		if _, ok := ownFieldsMap[joinFieldName]; ok {
			if field.Name != relation.FieldSchema.Name {
				joinFieldName = inflection.Singular(field.Name) + relField.Name
			} else {
				joinFieldName += "Reference"
			}
		}

		if len(joinReferences) > idx {
			joinFieldName = cases.Title(language.Und, cases.NoLower).String(joinReferences[idx])
		}

		referFieldsMap[joinFieldName] = relField

		if _, ok := fieldsMap[joinFieldName]; !ok {
			fieldsMap[joinFieldName] = relField
			joinTableFields = append(joinTableFields, reflect.StructField{
				Name:    joinFieldName,
				PkgPath: relField.StructField.PkgPath,
				Type:    relField.StructField.Type,
				Tag: removeSettingFromTag(appendSettingFromTag(relField.StructField.Tag, "primaryKey"),
					"column", "autoincrement", "index", "unique", "uniqueindex"),
			})
		}
	}

	joinTableFields = append(joinTableFields, reflect.StructField{
		Name: cases.Title(language.Und, cases.NoLower).String(schema.Name) + field.Name,
		Type: schema.ModelType,
		Tag:  `gorm:"-"`,
	})

	if relation.JoinTable, err = Parse(reflect.New(reflect.StructOf(joinTableFields)).Interface(), schema.cacheStore,
		schema.namer); err != nil {
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
			if copyableDataType(fieldsMap[f.Name].DataType) {
				f.DataType = fieldsMap[f.Name].DataType
			}
			f.GORMDataType = fieldsMap[f.Name].GORMDataType
			if f.Size == 0 {
				f.Size = fieldsMap[f.Name].Size
			}
			relation.JoinTable.PrimaryFields = append(relation.JoinTable.PrimaryFields, f)

			if of, ok := ownFieldsMap[f.Name]; ok {
				joinRel := relation.JoinTable.Relationships.Relations[relName]
				joinRel.Field = relation.Field
				joinRel.References = append(joinRel.References, &Reference{
					PrimaryKey: of,
					ForeignKey: f,
				})

				relation.References = append(relation.References, &Reference{
					PrimaryKey:    of,
					ForeignKey:    f,
					OwnPrimaryKey: true,
				})
			}

			if rf, ok := referFieldsMap[f.Name]; ok {
				joinRefRel := relation.JoinTable.Relationships.Relations[relRefName]
				if joinRefRel.Field == nil {
					joinRefRel.Field = relation.Field
				}
				joinRefRel.References = append(joinRefRel.References, &Reference{
					PrimaryKey: rf,
					ForeignKey: f,
				})

				relation.References = append(relation.References, &Reference{
					PrimaryKey: rf,
					ForeignKey: f,
				})
			}
		}
	}
}

// guessLevel tracks the fallback direction and embedded-owner scope used while
// inferring ordinary relationships.
type guessLevel int

const (
	// guessGuess starts relation inference using the model/field relationship to choose the
	// initial direction.
	guessGuess guessLevel = iota
	// guessBelongs looks for foreign keys on the association owner.
	guessBelongs
	// guessEmbeddedBelongs looks for foreign keys on the original embedded owner schema.
	guessEmbeddedBelongs
	// guessHas looks for foreign keys on the associated model that reference the owner.
	guessHas
	// guessEmbeddedHas looks for foreign keys on the associated model that reference the
	// original embedded owner.
	guessEmbeddedHas
)

// guessRelation matches explicit foreignKey/references tags or conventional key names,
// trying belongs-to, has, and embedded-owner alternatives as needed. On success it
// copies compatible key type metadata and appends ordered References; exhausted
// inference records an error asking for valid foreign keys or a scalar Scanner/Valuer
// type.
func (schema *Schema) guessRelation(relation *Relationship, field *Field, cgl guessLevel) {
	var (
		primaryFields, foreignFields []*Field
		primarySchema, foreignSchema = schema, relation.FieldSchema
		gl                           = cgl
	)

	if gl == guessGuess {
		if field.Schema == relation.FieldSchema {
			gl = guessBelongs
		} else {
			gl = guessHas
		}
	}

	reguessOrErr := func() {
		switch cgl {
		case guessGuess:
			schema.guessRelation(relation, field, guessBelongs)
		case guessBelongs:
			schema.guessRelation(relation, field, guessEmbeddedBelongs)
		case guessEmbeddedBelongs:
			schema.guessRelation(relation, field, guessHas)
		case guessHas:
			schema.guessRelation(relation, field, guessEmbeddedHas)
		// case guessEmbeddedHas:
		default:
			schema.err = fmt.Errorf("invalid field found for struct %v's field %s: define a valid foreign key for relations or implement the Valuer/Scanner interface",
				schema, field.Name)
		}
	}

	switch gl {
	case guessBelongs:
		primarySchema, foreignSchema = relation.FieldSchema, schema
	case guessEmbeddedBelongs:
		if field.OwnerSchema == nil {
			reguessOrErr()
			return
		}
		primarySchema, foreignSchema = relation.FieldSchema, field.OwnerSchema
	case guessHas:
	case guessEmbeddedHas:
		if field.OwnerSchema == nil {
			reguessOrErr()
			return
		}
		primarySchema, foreignSchema = field.OwnerSchema, relation.FieldSchema
	}

	if len(relation.foreignKeys) > 0 {
		for _, foreignKey := range relation.foreignKeys {
			f := foreignSchema.LookUpField(foreignKey)
			if f == nil {
				reguessOrErr()
				return
			}
			foreignFields = append(foreignFields, f)
		}
	} else {
		primarySchemaName := primarySchema.Name
		if primarySchemaName == "" {
			primarySchemaName = relation.FieldSchema.Name
		}

		if len(relation.primaryKeys) > 0 {
			for _, primaryKey := range relation.primaryKeys {
				if f := primarySchema.LookUpField(primaryKey); f != nil {
					primaryFields = append(primaryFields, f)
				}
			}
		} else {
			primaryFields = primarySchema.PrimaryFields
		}

	primaryFieldLoop:
		for _, primaryField := range primaryFields {
			lookUpName := primarySchemaName + primaryField.Name
			if gl == guessBelongs {
				lookUpName = field.Name + primaryField.Name
			}

			lookUpNames := []string{lookUpName}
			if len(primaryFields) == 1 {
				lookUpNames = append(lookUpNames, strings.TrimSuffix(lookUpName, primaryField.Name)+"ID",
					strings.TrimSuffix(lookUpName, primaryField.Name)+"Id", schema.namer.ColumnName(foreignSchema.Table,
						strings.TrimSuffix(lookUpName, primaryField.Name)+"ID"))
			}

			for _, name := range lookUpNames {
				if f := foreignSchema.LookUpFieldByBindName(field.BindNames, name); f != nil {
					foreignFields = append(foreignFields, f)
					primaryFields = append(primaryFields, primaryField)
					continue primaryFieldLoop
				}
			}
			for _, name := range lookUpNames {
				if f := foreignSchema.LookUpField(name); f != nil {
					foreignFields = append(foreignFields, f)
					primaryFields = append(primaryFields, primaryField)
					continue primaryFieldLoop
				}
			}
		}
	}

	switch {
	case len(foreignFields) == 0:
		reguessOrErr()
		return
	case len(relation.primaryKeys) > 0:
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
	case len(primaryFields) == 0:
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
		schema.Relationships.Mux.Lock()
		if schema != foreignField.Schema {
			foreignField.Schema.Relationships.Mux.Lock()
		}
		if copyableDataType(primaryFields[idx].DataType) {
			foreignField.DataType = primaryFields[idx].DataType
		}
		foreignField.GORMDataType = primaryFields[idx].GORMDataType
		if foreignField.Size == 0 {
			foreignField.Size = primaryFields[idx].Size
		}
		schema.Relationships.Mux.Unlock()
		if schema != foreignField.Schema {
			foreignField.Schema.Relationships.Mux.Unlock()
		}

		relation.References = append(relation.References, &Reference{
			PrimaryKey:    primaryFields[idx],
			ForeignKey:    foreignField,
			OwnPrimaryKey: (schema == primarySchema && gl == guessHas) || (field.OwnerSchema == primarySchema && gl == guessEmbeddedHas),
		})
	}

	if gl == guessHas || gl == guessEmbeddedHas {
		relation.Type = has
	} else {
		relation.Type = BelongsTo
	}
}

// Constraint describes a foreign-key constraint for migration, including the table that
// owns it, referenced table, paired columns, and referential actions. It does not
// execute or validate DDL.
type Constraint struct {
	// Name is the foreign-key constraint identifier, explicit or generated by
	// Namer.RelationshipFKName.
	Name string
	// Field is the association field from which this constraint was inferred.
	Field *Field
	// Schema is the table containing ForeignKeys; it can differ from the association owner
	// for has-one and has-many relations.
	Schema *Schema
	// ForeignKeys lists the constrained columns, paired positionally with References.
	ForeignKeys []*Field
	// ReferenceSchema is the table containing the referenced keys.
	ReferenceSchema *Schema
	// References lists the referenced key columns, paired positionally with ForeignKeys.
	References []*Field
	// OnDelete is the trusted SQL action appended after ON DELETE, such as CASCADE or SET
	// NULL. It is not a parameterized value.
	OnDelete string
	// OnUpdate is the trusted SQL action appended after ON UPDATE. Database support and
	// validity are checked when DDL executes.
	OnUpdate string
}

// GetName returns the foreign-key constraint identifier.
func (constraint *Constraint) GetName() string { return constraint.Name }

// Build returns a SQL template and clause arguments for the foreign-key definition.
// Identifiers are represented as clause values for dialect quoting; OnDelete and
// OnUpdate are appended as trusted SQL. It neither executes SQL nor checks column
// cardinality.
func (constraint *Constraint) Build() (sql string, vars []interface{}) {
	sql = "CONSTRAINT ? FOREIGN KEY ? REFERENCES ??"
	if constraint.OnDelete != "" {
		sql += " ON DELETE " + constraint.OnDelete
	}

	if constraint.OnUpdate != "" {
		sql += " ON UPDATE " + constraint.OnUpdate
	}

	foreignKeys := make([]interface{}, 0, len(constraint.ForeignKeys))
	for _, field := range constraint.ForeignKeys {
		foreignKeys = append(foreignKeys, clause.Column{Name: field.DBName})
	}

	references := make([]interface{}, 0, len(constraint.References))
	for _, field := range constraint.References {
		references = append(references, clause.Column{Name: field.DBName})
	}
	vars = append(vars, clause.Table{Name: constraint.Name}, foreignKeys, clause.Table{Name: constraint.ReferenceSchema.Table}, references)
	return
}

// ParseConstraint derives a foreign-key migration constraint from relationship
// references and the constraint tag. It returns nil for constraint:- and for duplicate
// inverse belongs-to mappings. Fixed polymorphic discriminator references do not become
// foreign-key columns; the actual key direction determines the owning and referenced
// schemas.
func (rel *Relationship) ParseConstraint() *Constraint {
	str := rel.Field.TagSettings["CONSTRAINT"]
	if str == "-" {
		return nil
	}

	if rel.Type == BelongsTo {
		for _, r := range rel.FieldSchema.Relationships.Relations {
			if r != rel && r.FieldSchema == rel.Schema && len(rel.References) == len(r.References) {
				matched := true
				for idx, ref := range r.References {
					if rel.References[idx].PrimaryKey != ref.PrimaryKey ||
						rel.References[idx].ForeignKey != ref.ForeignKey ||
						rel.References[idx].PrimaryValue != ref.PrimaryValue {
						matched = false
						break
					}
				}

				if matched {
					return nil
				}
			}
		}
	}

	var (
		name     string
		idx      = strings.IndexByte(str, ',')
		settings = ParseTagSetting(str, ",")
	)

	// optimize match english letters and midline
	// The following code is basically called in for.
	// In order to avoid the performance problems caused by repeated compilation of regular expressions,
	// it only needs to be done once outside, so optimization is done here.
	if idx != -1 && regEnLetterAndMidline.MatchString(str[0:idx]) {
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
		if ref.PrimaryKey != nil && (rel.JoinTable == nil || ref.OwnPrimaryKey) {
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

	return &constraint
}

// ToQueryConditions builds association predicates for the supplied owner model or
// collection: fixed discriminators, join-column equalities when needed, and an IN
// condition over extracted identity values. It performs no SQL and does not load rows.
// All-zero struct identities are excluded; an empty identity set remains an empty IN
// expression for the clause builder to handle.
func (rel *Relationship) ToQueryConditions(ctx context.Context, reflectValue reflect.Value) (conds []clause.Expression) {
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

	_, foreignValues := GetIdentityFieldValuesMap(ctx, reflectValue, foreignFields)
	column, values := ToQueryValues(table, relForeignKeys, foreignValues)

	conds = append(conds, clause.IN{Column: column, Values: values})
	return
}

// copyableDataType rejects explicit type strings containing auto_increment or primary
// key, case-insensitively, so copying a key type to a foreign-key field does not copy
// those column behaviors.
func copyableDataType(str DataType) bool {
	lowerStr := strings.ToLower(string(str))
	for _, s := range []string{"auto_increment", "primary key"} {
		if strings.Contains(lowerStr, s) {
			return false
		}
	}
	return true
}
