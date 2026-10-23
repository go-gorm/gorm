package schema

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"path"
	"reflect"
	"strings"
	"sync"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// callbackType identifies a model hook method inspected during schema parsing.
type callbackType string

const (
	// callbackTypeBeforeCreate identifies the BeforeCreate model hook before insertion.
	// Hook availability is discovered here; callbacks control execution and transaction
	// timing.
	callbackTypeBeforeCreate callbackType = "BeforeCreate"
	// callbackTypeBeforeUpdate identifies the BeforeUpdate model hook before updating. Hook
	// availability is discovered here; callbacks control execution and transaction timing.
	callbackTypeBeforeUpdate callbackType = "BeforeUpdate"
	// callbackTypeAfterCreate identifies the AfterCreate model hook after insertion. Hook
	// availability is discovered here; callbacks control execution and transaction timing.
	callbackTypeAfterCreate callbackType = "AfterCreate"
	// callbackTypeAfterUpdate identifies the AfterUpdate model hook after updating. Hook
	// availability is discovered here; callbacks control execution and transaction timing.
	callbackTypeAfterUpdate callbackType = "AfterUpdate"
	// callbackTypeBeforeSave identifies the BeforeSave model hook before a create or
	// update. Hook availability is discovered here; callbacks control execution and
	// transaction timing.
	callbackTypeBeforeSave callbackType = "BeforeSave"
	// callbackTypeAfterSave identifies the AfterSave model hook after a create or update.
	// Hook availability is discovered here; callbacks control execution and transaction
	// timing.
	callbackTypeAfterSave callbackType = "AfterSave"
	// callbackTypeBeforeDelete identifies the BeforeDelete model hook before deletion. Hook
	// availability is discovered here; callbacks control execution and transaction timing.
	callbackTypeBeforeDelete callbackType = "BeforeDelete"
	// callbackTypeAfterDelete identifies the AfterDelete model hook after deletion. Hook
	// availability is discovered here; callbacks control execution and transaction timing.
	callbackTypeAfterDelete callbackType = "AfterDelete"
	// callbackTypeAfterFind identifies the AfterFind model hook after query results are
	// scanned. Hook availability is discovered here; callbacks control execution and
	// transaction timing.
	callbackTypeAfterFind callbackType = "AfterFind"
)

// ErrUnsupportedDataType is wrapped when Parse cannot resolve a struct model type. Use
// errors.Is to recognize this error; maps and scalar values are not model schemas.
var ErrUnsupportedDataType = errors.New("unsupported data type")

// Schema is cached metadata for a Go struct model: its table, fields, keys,
// relationships, field-provided clauses, and available hooks. It describes a model, not
// live database introspection; Parse does not execute SQL or verify that tables exist.
//
// Obtain a Schema through Parse or a GORM Statement. Treat returned metadata as shared:
// concurrent operations may reuse the same pointers, and arbitrary mutation is not
// synchronized.
type Schema struct {
	// Name is the Go model type name; dynamically constructed anonymous types may have an
	// empty name.
	Name string
	// ModelType is the underlying struct type after removing pointer and collection
	// wrappers.
	ModelType reflect.Type
	// Table is the resolved database table name, including a naming-strategy prefix or
	// explicit table override.
	Table string
	// PrioritizedPrimaryField is the preferred key for single-key conventions: ID when
	// eligible, the sole primary key, or an auto-increment key in a composite key. It can
	// be nil; it does not replace PrimaryFields.
	PrioritizedPrimaryField *Field
	// DBNames lists unique mapped column names in discovery order. Relationship fields
	// without a column are excluded.
	DBNames []string
	// PrimaryFields contains all effective primary-key fields, including composite keys.
	PrimaryFields []*Field
	// PrimaryFieldDBNames contains the column names corresponding to PrimaryFields.
	PrimaryFieldDBNames []string
	// Fields contains parsed exported fields, with embedded structs flattened into their
	// constituent fields. It can include ignored and relationship fields.
	Fields []*Field
	// FieldsByName indexes fields by Go field name. Embedded-name collisions are resolved
	// during parsing; use FieldsByBindName for a full path.
	FieldsByName map[string]*Field
	// FieldsByBindName indexes dotted Go field paths such as Address.City, preserving
	// embedded-field identity.
	FieldsByBindName map[string]*Field // embedded fields is 'Embed.Field'
	// FieldsByDBName indexes the selected field for each database column. A permitted field
	// with a shorter embedding path takes precedence over a deeper collision.
	FieldsByDBName map[string]*Field
	// FieldsWithDefaultDBValue lists typed fields whose default is supplied by the database
	// rather than DefaultValueInterface, including generated integer primary keys. Create
	// callbacks use it for omission and RETURNING decisions.
	FieldsWithDefaultDBValue []*Field // fields with default value assigned by database
	// Relationships contains association metadata used for joins, preloading, association
	// saves, and migration.
	Relationships Relationships
	// CreateClauses contains clauses contributed by field types for create callbacks.
	CreateClauses []clause.Interface
	// QueryClauses contains clauses contributed by field types for queries, such as
	// soft-delete filtering.
	QueryClauses []clause.Interface
	// UpdateClauses contains clauses contributed by field types for update callbacks.
	UpdateClauses []clause.Interface
	// DeleteClauses contains clauses contributed by field types for deletes, such as
	// converting a delete into a soft-delete update.
	DeleteClauses []clause.Interface
	// BeforeCreate and AfterCreate indicate correctly typed create hooks on the model
	// pointer. They do not indicate that hooks will run for a particular operation.
	BeforeCreate, AfterCreate bool
	// BeforeUpdate and AfterUpdate indicate correctly typed update hooks on the model
	// pointer.
	BeforeUpdate, AfterUpdate bool
	// BeforeDelete and AfterDelete indicate correctly typed delete hooks on the model
	// pointer.
	BeforeDelete, AfterDelete bool
	// BeforeSave and AfterSave indicate correctly typed save hooks, shared by create and
	// update callbacks.
	BeforeSave, AfterSave bool
	// AfterFind indicates a correctly typed query hook on the model pointer.
	AfterFind bool
	// err records schema construction failures and deferred metadata parse errors.
	err error
	// initialized is closed when ParseWithSpecialTableName finishes building this schema.
	// External cache hits wait for it; recursive relationship parsing uses getOrParse to
	// avoid waiting on itself.
	initialized chan struct{}
	// namer supplies stable naming rules used during parsing and later metadata lookups.
	namer Namer
	// cacheStore shares schemas across model types and special table names, and lets
	// recursive relationships resolve an in-progress schema.
	cacheStore *sync.Map
}

// String returns the qualified Go type name, or Name(Table) for an anonymous model. It
// is a diagnostic representation, not SQL.
func (schema *Schema) String() string {
	if schema.ModelType.Name() == "" {
		return fmt.Sprintf("%s(%s)", schema.Name, schema.Table)
	}
	return fmt.Sprintf("%s.%s", schema.ModelType.PkgPath(), schema.ModelType.Name())
}

// MakeSlice returns a reflect.Value holding a pointer to an empty slice of model
// pointers, with initial capacity 20. Association code can append records through its
// Elem.
func (schema *Schema) MakeSlice() reflect.Value {
	slice := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(schema.ModelType)), 0, 20)
	results := reflect.New(slice.Type())
	results.Elem().Set(slice)

	return results
}

// LookUpField resolves a database column name first, then a Go field name, then the
// result of Namer.ColumnName. It returns nil when none matches; dotted embedding paths
// belong to FieldsByBindName.
func (schema *Schema) LookUpField(name string) *Field {
	if field, ok := schema.FieldsByDBName[name]; ok {
		return field
	}
	if field, ok := schema.FieldsByName[name]; ok {
		return field
	}

	// Lookup field using namer-driven ColumnName
	if schema.namer == nil {
		return nil
	}
	namerColumnName := schema.namer.ColumnName(schema.Table, name)
	if field, ok := schema.FieldsByDBName[namerColumnName]; ok {
		return field
	}

	return nil
}

// LookUpFieldByBindName searches for name relative to the enclosing embedded paths in
// bindNames, starting at the nearest scope. For ["Address", "City"], a lookup for "ID"
// tries "Address.ID" before ".ID". It returns nil if no such path exists; callers
// needing a top-level fallback should also call LookUpField.
func (schema *Schema) LookUpFieldByBindName(bindNames []string, name string) *Field {
	for i := len(bindNames) - 1; i >= 0; i-- {
		find := strings.Join(bindNames[:i], ".") + "." + name
		if field, ok := schema.FieldsByBindName[find]; ok {
			return field
		}
	}
	return nil
}

// Tabler lets a model provide a static table name. Parse calls TableName on a new model
// pointer and caches the result, so it must not depend on individual row values. Use
// DB.Table or a scope for per-operation table routing.
type Tabler interface {
	// TableName returns the model table name before any explicit special-table override.
	TableName() string
}

// TablerWithNamer lets a model derive its static table name from the configured Namer.
// Tabler takes precedence when both are applicable; results are cached like other
// schema metadata.
type TablerWithNamer interface {
	// TableName returns the model table name using the supplied naming strategy.
	TableName(Namer) string
}

// callbackTypes lists the hook methods checked on a model pointer. Parsing accepts
// func(*gorm.DB) error from the GORM package and warns about incompatible signatures.
var callbackTypes = []callbackType{
	callbackTypeBeforeCreate, callbackTypeAfterCreate,
	callbackTypeBeforeUpdate, callbackTypeAfterUpdate,
	callbackTypeBeforeSave, callbackTypeAfterSave,
	callbackTypeBeforeDelete, callbackTypeAfterDelete,
	callbackTypeAfterFind,
}

// Parse resolves dest to struct metadata using cacheStore and namer. It accepts models,
// pointers, and slices or arrays of models; it returns an error wrapping
// ErrUnsupportedDataType for unsupported types. cacheStore and namer must be non-nil.
//
// The cache is keyed by model type, not by naming strategy. Reuse a cache only with
// compatible, stable naming rules. Cache hits wait for schema initialization and return
// the shared metadata and its parse error. No database connection or SQL is involved.
//
// For example:
//
//	cache := &sync.Map{}
//	s, err := schema.Parse(&User{}, cache, schema.NamingStrategy{})
//
// Check err before using fields or relationships. Use ParseWithSpecialTableName when
// the schema itself must be associated with an explicit table.
func Parse(dest interface{}, cacheStore *sync.Map, namer Namer) (*Schema, error) {
	return ParseWithSpecialTableName(dest, cacheStore, namer, "")
}

// ParseWithSpecialTableName performs Parse with an optional table override. A nonempty
// specialTableName takes precedence over embedded naming, Tabler, TablerWithNamer, and
// Namer.TableName, and gets a separate cache entry per model type and table name.
//
// It flattens exported embedded fields, resolves column and primary-key precedence,
// initializes accessors and field clauses, validates hooks, then parses relationships.
// Recursive relationships can reference partially initialized cached metadata
// internally; ordinary callers wait for initialization. A non-nil returned schema may
// accompany an error and must not be used as successfully parsed metadata.
func ParseWithSpecialTableName(dest interface{}, cacheStore *sync.Map, namer Namer, specialTableName string) (*Schema, error) {
	if dest == nil {
		return nil, fmt.Errorf("%w: %+v", ErrUnsupportedDataType, dest)
	}

	modelType := reflect.ValueOf(dest).Type()
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		if modelType.Kind() == reflect.Interface {
			modelType = reflect.Indirect(reflect.ValueOf(dest)).Elem().Type()
		}

		for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}

		if modelType.Kind() != reflect.Struct {
			if modelType.PkgPath() == "" {
				return nil, fmt.Errorf("%w: %+v", ErrUnsupportedDataType, dest)
			}
			return nil, fmt.Errorf("%w: %s.%s", ErrUnsupportedDataType, modelType.PkgPath(), modelType.Name())
		}
	}

	// Cache the Schema for performance,
	// Use the modelType or modelType + schemaTable (if it present) as cache key.
	var schemaCacheKey interface{} = modelType
	if specialTableName != "" {
		schemaCacheKey = fmt.Sprintf("%p-%s", modelType, specialTableName)
	}

	// Load exist schema cache, return if exists
	if v, ok := cacheStore.Load(schemaCacheKey); ok {
		s := v.(*Schema)
		// Wait for the initialization of other goroutines to complete
		<-s.initialized
		return s, s.err
	}

	var tableName string
	modelValue := reflect.New(modelType)
	if specialTableName != "" {
		tableName = specialTableName
	} else if en, ok := namer.(embeddedNamer); ok {
		tableName = en.Table
	} else if tabler, ok := modelValue.Interface().(Tabler); ok {
		tableName = tabler.TableName()
	} else if tabler, ok := modelValue.Interface().(TablerWithNamer); ok {
		tableName = tabler.TableName(namer)
	} else {
		tableName = namer.TableName(modelType.Name())
	}

	schema := &Schema{
		Name:             modelType.Name(),
		ModelType:        modelType,
		Table:            tableName,
		DBNames:          make([]string, 0, 10),
		Fields:           make([]*Field, 0, 10),
		FieldsByName:     make(map[string]*Field, 10),
		FieldsByBindName: make(map[string]*Field, 10),
		FieldsByDBName:   make(map[string]*Field, 10),
		Relationships:    Relationships{Relations: map[string]*Relationship{}},
		cacheStore:       cacheStore,
		namer:            namer,
		initialized:      make(chan struct{}),
	}
	// When the schema initialization is completed, the channel will be closed
	defer close(schema.initialized)

	// Load exist schema cache, return if exists
	if v, ok := cacheStore.Load(schemaCacheKey); ok {
		s := v.(*Schema)
		// Wait for the initialization of other goroutines to complete
		<-s.initialized
		return s, s.err
	}

	for i := 0; i < modelType.NumField(); i++ {
		if fieldStruct := modelType.Field(i); ast.IsExported(fieldStruct.Name) {
			if field := schema.ParseField(fieldStruct); field.EmbeddedSchema != nil {
				schema.Fields = append(schema.Fields, field.EmbeddedSchema.Fields...)
			} else {
				schema.Fields = append(schema.Fields, field)
			}
		}
	}

	for _, field := range schema.Fields {
		if field.DBName == "" && field.DataType != "" {
			field.DBName = namer.ColumnName(schema.Table, field.Name)
		}

		bindName := field.BindName()
		if field.DBName != "" {
			// nonexistence or shortest path or first appear prioritized if has permission
			if v, ok := schema.FieldsByDBName[field.DBName]; !ok || ((field.Creatable || field.Updatable || field.Readable) && len(field.BindNames) < len(v.BindNames)) {
				if _, ok := schema.FieldsByDBName[field.DBName]; !ok {
					schema.DBNames = append(schema.DBNames, field.DBName)
				}
				schema.FieldsByDBName[field.DBName] = field
				schema.FieldsByName[field.Name] = field
				schema.FieldsByBindName[bindName] = field

				if v != nil && v.PrimaryKey {
					// remove the existing primary key field
					for idx, f := range schema.PrimaryFields {
						if f.DBName == v.DBName {
							schema.PrimaryFields = append(schema.PrimaryFields[0:idx], schema.PrimaryFields[idx+1:]...)
						}
					}
				}

				if field.PrimaryKey {
					schema.PrimaryFields = append(schema.PrimaryFields, field)
				}
			}
		}

		if of, ok := schema.FieldsByName[field.Name]; !ok || of.TagSettings["-"] == "-" {
			schema.FieldsByName[field.Name] = field
		}
		if of, ok := schema.FieldsByBindName[bindName]; !ok || of.TagSettings["-"] == "-" {
			schema.FieldsByBindName[bindName] = field
		}

		field.setupValuerAndSetter(modelType)
	}

	prioritizedPrimaryField := schema.LookUpField("id")
	if prioritizedPrimaryField == nil {
		prioritizedPrimaryField = schema.LookUpField("ID")
	}

	if prioritizedPrimaryField != nil {
		if prioritizedPrimaryField.PrimaryKey {
			schema.PrioritizedPrimaryField = prioritizedPrimaryField
		} else if len(schema.PrimaryFields) == 0 {
			prioritizedPrimaryField.PrimaryKey = true
			schema.PrioritizedPrimaryField = prioritizedPrimaryField
			schema.PrimaryFields = append(schema.PrimaryFields, prioritizedPrimaryField)
		}
	}

	if schema.PrioritizedPrimaryField == nil {
		if len(schema.PrimaryFields) == 1 {
			schema.PrioritizedPrimaryField = schema.PrimaryFields[0]
		} else if len(schema.PrimaryFields) > 1 {
			// If there are multiple primary keys, the AUTOINCREMENT field is prioritized
			for _, field := range schema.PrimaryFields {
				if field.AutoIncrement {
					schema.PrioritizedPrimaryField = field
					break
				}
			}
		}
	}

	for _, field := range schema.PrimaryFields {
		schema.PrimaryFieldDBNames = append(schema.PrimaryFieldDBNames, field.DBName)
	}

	_, embedded := schema.cacheStore.Load(embeddedCacheKey)
	relationshipFields := []*Field{}
	for _, field := range schema.Fields {
		if field.DataType != "" && field.HasDefaultValue && field.DefaultValueInterface == nil {
			schema.FieldsWithDefaultDBValue = append(schema.FieldsWithDefaultDBValue, field)
		}

		if !embedded {
			if field.DataType == "" && field.GORMDataType == "" && (field.Creatable || field.Updatable || field.Readable) {
				relationshipFields = append(relationshipFields, field)
				schema.FieldsByName[field.Name] = field
				schema.FieldsByBindName[field.BindName()] = field
			}

			fieldValue := reflect.New(field.IndirectFieldType).Interface()
			if fc, ok := fieldValue.(CreateClausesInterface); ok {
				field.Schema.CreateClauses = append(field.Schema.CreateClauses, fc.CreateClauses(field)...)
			}

			if fc, ok := fieldValue.(QueryClausesInterface); ok {
				field.Schema.QueryClauses = append(field.Schema.QueryClauses, fc.QueryClauses(field)...)
			}

			if fc, ok := fieldValue.(UpdateClausesInterface); ok {
				field.Schema.UpdateClauses = append(field.Schema.UpdateClauses, fc.UpdateClauses(field)...)
			}

			if fc, ok := fieldValue.(DeleteClausesInterface); ok {
				field.Schema.DeleteClauses = append(field.Schema.DeleteClauses, fc.DeleteClauses(field)...)
			}
		}
	}

	if field := schema.PrioritizedPrimaryField; field != nil {
		switch field.GORMDataType {
		case Int, Uint:
			if _, ok := field.TagSettings["AUTOINCREMENT"]; !ok {
				if !field.HasDefaultValue || field.DefaultValueInterface != nil {
					schema.FieldsWithDefaultDBValue = append(schema.FieldsWithDefaultDBValue, field)
				}

				field.HasDefaultValue = true
				field.AutoIncrement = true
			}
		}
	}

	// Cache the schema
	if v, loaded := cacheStore.LoadOrStore(schemaCacheKey, schema); loaded {
		s := v.(*Schema)
		// Wait for the initialization of other goroutines to complete
		<-s.initialized
		return s, s.err
	}

	defer func() {
		if schema.err != nil {
			logger.Default.Error(context.Background(), schema.err.Error())
			cacheStore.Delete(modelType)
		}
	}()

	for _, cbName := range callbackTypes {
		if methodValue := modelValue.MethodByName(string(cbName)); methodValue.IsValid() {
			switch methodValue.Type().String() {
			case "func(*gorm.DB) error":
				expectedPkgPath := path.Dir(reflect.TypeOf(schema).Elem().PkgPath())
				if inVarPkg := methodValue.Type().In(0).Elem().PkgPath(); inVarPkg == expectedPkgPath {
					reflect.Indirect(reflect.ValueOf(schema)).FieldByName(string(cbName)).SetBool(true)
				} else {
					logger.Default.Warn(context.Background(), "In model %v, the hook function `%v(*gorm.DB) error` has an incorrect parameter type. The expected parameter type is `%v`, but the provided type is `%v`.", schema, cbName, expectedPkgPath, inVarPkg)
					// PASS
				}
			default:
				logger.Default.Warn(context.Background(), "Model %v don't match %vInterface, should be `%v(*gorm.DB) error`. Please see https://gorm.io/docs/hooks.html", schema, cbName, cbName)
			}
		}
	}

	// parse relationships
	for _, field := range relationshipFields {
		if schema.parseRelation(field); schema.err != nil {
			return schema, schema.err
		}
	}

	return schema, schema.err
}

// getOrParse resolves a struct model from the shared cache without waiting for its
// initialized channel, allowing cyclic relationship graphs to refer to a schema under
// construction. It is an internal recursion helper; public callers should use Parse to
// wait for complete metadata.
func getOrParse(dest interface{}, cacheStore *sync.Map, namer Namer) (*Schema, error) {
	modelType := reflect.ValueOf(dest).Type()

	if modelType.Kind() != reflect.Struct {
		for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}

		if modelType.Kind() != reflect.Struct {
			if modelType.PkgPath() == "" {
				return nil, fmt.Errorf("%w: %+v", ErrUnsupportedDataType, dest)
			}
			return nil, fmt.Errorf("%w: %s.%s", ErrUnsupportedDataType, modelType.PkgPath(), modelType.Name())
		}
	}

	if v, ok := cacheStore.Load(modelType); ok {
		return v.(*Schema), nil
	}

	return Parse(dest, cacheStore, namer)
}
