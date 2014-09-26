package gorm

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"
	"time"

	"reflect"
	"regexp"
)

type Scope struct {
	Value         interface{}
	indirectValue *reflect.Value
	Search        *search
	Sql           string
	SqlVars       []interface{}
	db            *DB
	skipLeft      bool
	primaryKey    string
	instanceId    string
	fields        map[string]*Field
}

func (scope *Scope) IndirectValue() reflect.Value {
	if scope.indirectValue == nil {
		value := reflect.Indirect(reflect.ValueOf(scope.Value))
		scope.indirectValue = &value
	}
	return *scope.indirectValue
}

// NewScope create scope for callbacks, including DB's search information
func (db *DB) NewScope(value interface{}) *Scope {
	// reflectKind := reflect.ValueOf(value).Kind()
	// if !((reflectKind == reflect.Invalid) || (reflectKind == reflect.Ptr)) {
	// 	fmt.Printf("%v %v\n", fileWithLineNum(), "using unaddressable value")
	// }
	db.Value = value
	return &Scope{db: db, Search: db.search, Value: value}
}

// New create a new Scope without search information
func (scope *Scope) New(value interface{}) *Scope {
	return &Scope{db: scope.db, Search: &search{}, Value: value}
}

// NewDB create a new DB without search information
func (scope *Scope) NewDB() *DB {
	return scope.db.new()
}

// DB get *sql.DB
func (scope *Scope) DB() sqlCommon {
	return scope.db.db
}

// SkipLeft skip remaining callbacks
func (scope *Scope) SkipLeft() {
	scope.skipLeft = true
}

// Quote used to quote database column name according to database dialect
func (scope *Scope) Quote(str string) string {
	return scope.Dialect().Quote(str)
}

// Dialect get dialect
func (scope *Scope) Dialect() Dialect {
	return scope.db.parent.dialect
}

// Err write error
func (scope *Scope) Err(err error) error {
	if err != nil {
		scope.db.err(err)
	}
	return err
}

// Log print log message
func (scope *Scope) Log(v ...interface{}) {
	scope.db.log(v...)
}

// HasError check if there are any error
func (scope *Scope) HasError() bool {
	return scope.db.Error != nil
}

// PrimaryKey get the primary key's column name
func (scope *Scope) PrimaryKey() string {
	if scope.primaryKey != "" {
		return scope.primaryKey
	}

	var indirectValue = scope.IndirectValue()

	clone := scope
	if indirectValue.Kind() == reflect.Slice {
		clone = scope.New(reflect.New(indirectValue.Type().Elem()).Elem().Interface())
	}

	for _, field := range clone.Fields() {
		if field.IsPrimaryKey {
			scope.primaryKey = field.DBName
			break
		}
	}

	return scope.primaryKey
}

// PrimaryKeyZero check the primary key is blank or not
func (scope *Scope) PrimaryKeyZero() bool {
	return isBlank(reflect.ValueOf(scope.PrimaryKeyValue()))
}

// PrimaryKeyValue get the primary key's value
func (scope *Scope) PrimaryKeyValue() interface{} {
	if scope.IndirectValue().Kind() == reflect.Struct {
		if field := scope.IndirectValue().FieldByName(SnakeToUpperCamel(scope.PrimaryKey())); field.IsValid() {
			return field.Interface()
		}
	}
	return 0
}

// HasColumn to check if has column
func (scope *Scope) HasColumn(column string) bool {
	clone := scope
	if scope.IndirectValue().Kind() == reflect.Slice {
		value := reflect.New(scope.IndirectValue().Type().Elem()).Interface()
		clone = scope.New(value)
	}

	for _, field := range clone.Fields(false) {
		if field.Name == column || field.DBName == column {
			return true
		}
	}
	return false
}

// FieldValueByName to get column's value and existence
func (scope *Scope) FieldValueByName(name string) (interface{}, bool) {
	return FieldValueByName(name, scope.Value)
}

// SetColumn to set the column's value
func (scope *Scope) SetColumn(column interface{}, value interface{}) bool {
	if field, ok := column.(*Field); ok {
		return field.Set(value)
	} else if str, ok := column.(string); ok {
		if scope.Value == nil {
			return false
		}
		for _, field := range scope.Fields() {
			if field.Name == str || field.DBName == str {
				return field.Set(value)
			}
		}
	}
	return false
}

// CallMethod invoke method with necessary argument
func (scope *Scope) CallMethod(name string) {
	if scope.Value == nil {
		return
	}

	call := func(value interface{}) {
		if fm := reflect.ValueOf(value).MethodByName(name); fm.IsValid() {
			fi := fm.Interface()
			if f, ok := fi.(func()); ok {
				f()
			} else if f, ok := fi.(func(s *Scope)); ok {
				f(scope)
			} else if f, ok := fi.(func(s *DB)); ok {
				f(scope.db.new())
			} else if f, ok := fi.(func() error); ok {
				scope.Err(f())
			} else if f, ok := fi.(func(s *Scope) error); ok {
				scope.Err(f(scope))
			} else if f, ok := fi.(func(s *DB) error); ok {
				scope.Err(f(scope.db.new()))
			} else {
				scope.Err(errors.New(fmt.Sprintf("unsupported function %v", name)))
			}
		}
	}

	if values := scope.IndirectValue(); values.Kind() == reflect.Slice {
		for i := 0; i < values.Len(); i++ {
			call(values.Index(i).Addr().Interface())
		}
	} else {
		call(scope.Value)
	}
}

// AddToVars add value as sql's vars, gorm will escape them
func (scope *Scope) AddToVars(value interface{}) string {
	scope.SqlVars = append(scope.SqlVars, value)
	return scope.Dialect().BinVar(len(scope.SqlVars))
}

// TableName get table name
var pluralMapKeys = []*regexp.Regexp{regexp.MustCompile("ch$"), regexp.MustCompile("ss$"), regexp.MustCompile("sh$"), regexp.MustCompile("day$"), regexp.MustCompile("y$"), regexp.MustCompile("x$"), regexp.MustCompile("([^s])s?$")}
var pluralMapValues = []string{"ches", "sses", "shes", "days", "ies", "xes", "${1}s"}

func (scope *Scope) TableName() string {
	if scope.Search != nil && len(scope.Search.TableName) > 0 {
		return scope.Search.TableName
	} else {
		if scope.Value == nil {
			scope.Err(errors.New("can't get table name"))
			return ""
		}

		data := scope.IndirectValue()
		if data.Kind() == reflect.Slice {
			elem := data.Type().Elem()
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			data = reflect.New(elem).Elem()
		}

		if fm := data.MethodByName("TableName"); fm.IsValid() {
			if v := fm.Call([]reflect.Value{}); len(v) > 0 {
				if result, ok := v[0].Interface().(string); ok {
					return result
				}
			}
		}

		str := ToSnake(data.Type().Name())

		if scope.db == nil || !scope.db.parent.singularTable {
			for index, reg := range pluralMapKeys {
				if reg.MatchString(str) {
					return reg.ReplaceAllString(str, pluralMapValues[index])
				}
			}
		}

		return str
	}
}

func (scope *Scope) QuotedTableName() string {
	if scope.Search != nil && len(scope.Search.TableName) > 0 {
		return scope.Search.TableName
	} else {
		keys := strings.Split(scope.TableName(), ".")
		for i, v := range keys {
			keys[i] = scope.Quote(v)
		}
		return strings.Join(keys, ".")
	}
}

// CombinedConditionSql get combined condition sql
func (scope *Scope) CombinedConditionSql() string {
	return scope.joinsSql() + scope.whereSql() + scope.groupSql() +
		scope.havingSql() + scope.orderSql() + scope.limitSql() + scope.offsetSql()
}

func (scope *Scope) FieldByName(name string) (field *Field, ok bool) {
	for _, field := range scope.Fields() {
		if field.Name == name {
			return field, true
		}
	}
	return nil, false
}

func (scope *Scope) fieldFromStruct(fieldStruct reflect.StructField, withRelation bool) []*Field {
	var field Field
	field.Name = fieldStruct.Name
	field.DBName = ToSnake(fieldStruct.Name)

	value := scope.IndirectValue().FieldByName(fieldStruct.Name)
	indirectValue := reflect.Indirect(value)
	field.Field = value
	field.IsBlank = isBlank(value)

	// Search for primary key tag identifier
	settings := parseTagSetting(fieldStruct.Tag.Get("gorm"))
	if _, ok := settings["PRIMARY_KEY"]; ok {
		field.IsPrimaryKey = true
	}

	field.Tag = fieldStruct.Tag

	tagIdentifier := "sql"
	if scope.db != nil {
		tagIdentifier = scope.db.parent.tagIdentifier
	}
	if fieldStruct.Tag.Get(tagIdentifier) == "-" {
		field.IsIgnored = true
	}

	if !field.IsIgnored {
		// parse association
		if !indirectValue.IsValid() {
			indirectValue = reflect.New(value.Type())
		}
		typ := indirectValue.Type()
		scopeTyp := scope.IndirectValue().Type()

		foreignKey := SnakeToUpperCamel(settings["FOREIGNKEY"])
		associationForeignKey := SnakeToUpperCamel(settings["ASSOCIATIONFOREIGNKEY"])
		many2many := settings["MANY2MANY"]

		switch indirectValue.Kind() {
		case reflect.Slice:
			typ = typ.Elem()

			if (typ.Kind() == reflect.Struct) && withRelation {
				if foreignKey == "" {
					foreignKey = scopeTyp.Name() + "Id"
				}
				if associationForeignKey == "" {
					associationForeignKey = typ.Name() + "Id"
				}

				// if not many to many, foreign key could be null
				if many2many == "" {
					if !reflect.New(typ).Elem().FieldByName(foreignKey).IsValid() {
						foreignKey = ""
					}
				}

				field.Relationship = &relationship{
					JoinTable:             many2many,
					ForeignKey:            foreignKey,
					AssociationForeignKey: associationForeignKey,
					Kind: "has_many",
				}

				if many2many != "" {
					field.Relationship.Kind = "many_to_many"
				}
			} else {
				field.IsNormal = true
			}
		case reflect.Struct:
			if field.IsTime() || field.IsScanner() {
				field.IsNormal = true
			} else if _, ok := settings["EMBEDDED"]; ok || fieldStruct.Anonymous {
				var fields []*Field
				if field.Field.CanAddr() {
					for _, field := range scope.New(field.Field.Addr().Interface()).Fields() {
						field.DBName = field.DBName
						fields = append(fields, field)
					}
				}
				return fields
			} else if withRelation {
				var belongsToForeignKey, hasOneForeignKey, kind string

				if foreignKey == "" {
					belongsToForeignKey = field.Name + "Id"
					hasOneForeignKey = scopeTyp.Name() + "Id"
				} else {
					belongsToForeignKey = foreignKey
					hasOneForeignKey = foreignKey
				}

				if scope.HasColumn(belongsToForeignKey) {
					foreignKey = belongsToForeignKey
					kind = "belongs_to"
				} else {
					foreignKey = hasOneForeignKey
					kind = "has_one"
				}

				field.Relationship = &relationship{ForeignKey: foreignKey, Kind: kind}
			}
		default:
			field.IsNormal = true
		}
	}
	return []*Field{&field}
}

// Fields get value's fields
func (scope *Scope) Fields(noRelations ...bool) map[string]*Field {
	if scope.fields != nil {
		return scope.fields
	}
	var withRelation = len(noRelations) == 0

	var fields = map[string]*Field{}
	if scope.IndirectValue().IsValid() && scope.IndirectValue().Kind() == reflect.Struct {
		scopeTyp := scope.IndirectValue().Type()
		var hasPrimaryKey = false
		for i := 0; i < scopeTyp.NumField(); i++ {
			fieldStruct := scopeTyp.Field(i)
			if !ast.IsExported(fieldStruct.Name) {
				continue
			}
			for _, field := range scope.fieldFromStruct(fieldStruct, withRelation) {
				if field.IsPrimaryKey {
					hasPrimaryKey = true
				}
				if _, ok := fields[field.DBName]; ok {
					panic(fmt.Sprintf("Duplicated column name for %v (%v)\n", scope.typeName(), fileWithLineNum()))
				} else {
					fields[field.DBName] = field
				}
			}
		}

		if !hasPrimaryKey {
			if field, ok := fields["id"]; ok {
				field.IsPrimaryKey = true
			}
		}
	}

	if withRelation {
		scope.fields = fields
	}

	return fields
}

// Raw set sql
func (scope *Scope) Raw(sql string) *Scope {
	scope.Sql = strings.Replace(sql, "$$", "?", -1)
	return scope
}

// Exec invoke sql
func (scope *Scope) Exec() *Scope {
	defer scope.Trace(NowFunc())

	if !scope.HasError() {
		result, err := scope.DB().Exec(scope.Sql, scope.SqlVars...)
		if scope.Err(err) == nil {
			if count, err := result.RowsAffected(); err == nil {
				scope.db.RowsAffected = count
			}
		}
	}
	return scope
}

// Set set value by name
func (scope *Scope) Set(name string, value interface{}) *Scope {
	scope.db.InstantSet(name, value)
	return scope
}

// Get get value by name
func (scope *Scope) Get(name string) (interface{}, bool) {
	return scope.db.Get(name)
}

// InstanceId get InstanceId for scope
func (scope *Scope) InstanceId() string {
	if scope.instanceId == "" {
		scope.instanceId = fmt.Sprintf("%v", &scope)
	}
	return scope.instanceId
}

func (scope *Scope) InstanceSet(name string, value interface{}) *Scope {
	return scope.Set(name+scope.InstanceId(), value)
}

func (scope *Scope) InstanceGet(name string) (interface{}, bool) {
	return scope.Get(name + scope.InstanceId())
}

// Trace print sql log
func (scope *Scope) Trace(t time.Time) {
	if len(scope.Sql) > 0 {
		scope.db.slog(scope.Sql, t, scope.SqlVars...)
	}
}

// Begin start a transaction
func (scope *Scope) Begin() *Scope {
	if db, ok := scope.DB().(sqlDb); ok {
		if tx, err := db.Begin(); err == nil {
			scope.db.db = interface{}(tx).(sqlCommon)
			scope.InstanceSet("gorm:started_transaction", true)
		}
	}
	return scope
}

// CommitOrRollback commit current transaction if there is no error, otherwise rollback it
func (scope *Scope) CommitOrRollback() *Scope {
	if _, ok := scope.InstanceGet("gorm:started_transaction"); ok {
		if db, ok := scope.db.db.(sqlTx); ok {
			if scope.HasError() {
				db.Rollback()
			} else {
				db.Commit()
			}
			scope.db.db = scope.db.parent.db
		}
	}
	return scope
}
