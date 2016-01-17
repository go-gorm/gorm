package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"reflect"
)

type Scope struct {
	Search          *search
	Value           interface{}
	Sql             string
	SqlVars         []interface{}
	db              *DB
	indirectValue   *reflect.Value
	instanceID      string
	primaryKeyField *Field
	skipLeft        bool
	fields          map[string]*Field
	selectAttrs     *[]string
}

func (scope *Scope) IndirectValue() reflect.Value {
	if scope.indirectValue == nil {
		value := reflect.Indirect(reflect.ValueOf(scope.Value))
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}
		scope.indirectValue = &value
	}
	return *scope.indirectValue
}

// New create a new Scope without search information
func (scope *Scope) New(value interface{}) *Scope {
	return &Scope{db: scope.NewDB(), Search: &search{}, Value: value}
}

// NewDB create a new DB without search information
func (scope *Scope) NewDB() *DB {
	if scope.db != nil {
		db := scope.db.clone()
		db.search = nil
		db.Value = nil
		return db
	}
	return nil
}

func (scope *Scope) DB() *DB {
	return scope.db
}

// SqlDB return *sql.DB
func (scope *Scope) SqlDB() sqlCommon {
	return scope.db.db
}

// SkipLeft skip remaining callbacks
func (scope *Scope) SkipLeft() {
	scope.skipLeft = true
}

// Quote used to quote database column name according to database dialect
func (scope *Scope) Quote(str string) string {
	if strings.Index(str, ".") != -1 {
		newStrs := []string{}
		for _, str := range strings.Split(str, ".") {
			newStrs = append(newStrs, scope.Dialect().Quote(str))
		}
		return strings.Join(newStrs, ".")
	}

	return scope.Dialect().Quote(str)
}

func (scope *Scope) quoteIfPossible(str string) string {
	if regexp.MustCompile("^[a-zA-Z]+(.[a-zA-Z]+)*$").MatchString(str) {
		return scope.Quote(str)
	}
	return str
}

// Dialect get dialect
func (scope *Scope) Dialect() Dialect {
	return scope.db.parent.dialect
}

// Err write error
func (scope *Scope) Err(err error) error {
	if err != nil {
		scope.db.AddError(err)
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

func (scope *Scope) PrimaryFields() []*Field {
	var fields = []*Field{}
	for _, field := range scope.GetModelStruct().PrimaryFields {
		fields = append(fields, scope.Fields()[field.DBName])
	}
	return fields
}

func (scope *Scope) PrimaryField() *Field {
	if primaryFields := scope.GetModelStruct().PrimaryFields; len(primaryFields) > 0 {
		if len(primaryFields) > 1 {
			if field, ok := scope.Fields()["id"]; ok {
				return field
			}
		}
		return scope.Fields()[primaryFields[0].DBName]
	}
	return nil
}

// PrimaryKey get the primary key's column name
func (scope *Scope) PrimaryKey() string {
	if field := scope.PrimaryField(); field != nil {
		return field.DBName
	}
	return ""
}

// PrimaryKeyZero check the primary key is blank or not
func (scope *Scope) PrimaryKeyZero() bool {
	field := scope.PrimaryField()
	return field == nil || field.IsBlank
}

// PrimaryKeyValue get the primary key's value
func (scope *Scope) PrimaryKeyValue() interface{} {
	if field := scope.PrimaryField(); field != nil && field.Field.IsValid() {
		return field.Field.Interface()
	}
	return 0
}

// HasColumn to check if has column
func (scope *Scope) HasColumn(column string) bool {
	for _, field := range scope.GetStructFields() {
		if field.IsNormal && (field.Name == column || field.DBName == column) {
			return true
		}
	}
	return false
}

// SetColumn to set the column's value
func (scope *Scope) SetColumn(column interface{}, value interface{}) error {
	if field, ok := column.(*Field); ok {
		return field.Set(value)
	} else if name, ok := column.(string); ok {

		if field, ok := scope.Fields()[name]; ok {
			return field.Set(value)
		}

		dbName := ToDBName(name)
		if field, ok := scope.Fields()[dbName]; ok {
			return field.Set(value)
		}

		if field, ok := scope.FieldByName(name); ok {
			return field.Set(value)
		}
	}
	return errors.New("could not convert column to field")
}

func (scope *Scope) callMethod(methodName string, reflectValue reflect.Value) {
	if reflectValue.CanAddr() {
		reflectValue = reflectValue.Addr()
	}

	if methodValue := reflectValue.MethodByName(methodName); methodValue.IsValid() {
		switch method := methodValue.Interface().(type) {
		case func():
			method()
		case func(*Scope):
			method(scope)
		case func(*DB):
			newDB := scope.NewDB()
			method(newDB)
			scope.Err(newDB.Error)
		case func() error:
			scope.Err(method())
		case func(*Scope) error:
			scope.Err(method(scope))
		case func(*DB) error:
			newDB := scope.NewDB()
			scope.Err(method(newDB))
			scope.Err(newDB.Error)
		default:
			scope.Err(fmt.Errorf("unsupported function %v", methodName))
		}
	}
}

// CallMethod call scope value's method, if it is a slice, will call value's method one by one
func (scope *Scope) CallMethod(methodName string) {
	if scope.Value == nil {
		return
	}

	if indirectScopeValue := scope.IndirectValue(); indirectScopeValue.Kind() == reflect.Slice {
		for i := 0; i < indirectScopeValue.Len(); i++ {
			scope.callMethod(methodName, indirectScopeValue.Index(i))
		}
	} else {
		scope.callMethod(methodName, indirectScopeValue)
	}
}

// AddToVars add value as sql's vars, gorm will escape them
func (scope *Scope) AddToVars(value interface{}) string {
	if expr, ok := value.(*expr); ok {
		exp := expr.expr
		for _, arg := range expr.args {
			exp = strings.Replace(exp, "?", scope.AddToVars(arg), 1)
		}
		return exp
	}

	scope.SqlVars = append(scope.SqlVars, value)
	return scope.Dialect().BinVar(len(scope.SqlVars))
}

type tabler interface {
	TableName() string
}

type dbTabler interface {
	TableName(*DB) string
}

// TableName return table name
func (scope *Scope) TableName() string {
	if scope.Search != nil && len(scope.Search.tableName) > 0 {
		return scope.Search.tableName
	}

	if tabler, ok := scope.Value.(tabler); ok {
		return tabler.TableName()
	}

	if tabler, ok := scope.Value.(dbTabler); ok {
		return tabler.TableName(scope.db)
	}

	return scope.GetModelStruct().TableName(scope.db.Model(scope.Value))
}

// QuotedTableName return quoted table name
func (scope *Scope) QuotedTableName() (name string) {
	if scope.Search != nil && len(scope.Search.tableName) > 0 {
		if strings.Index(scope.Search.tableName, " ") != -1 {
			return scope.Search.tableName
		}
		return scope.Quote(scope.Search.tableName)
	}

	return scope.Quote(scope.TableName())
}

// CombinedConditionSql get combined condition sql
func (scope *Scope) CombinedConditionSql() string {
	return scope.joinsSql() + scope.whereSql() + scope.groupSql() +
		scope.havingSql() + scope.orderSql() + scope.limitSql() + scope.offsetSql()
}

// FieldByName find gorm.Field with name and db name
func (scope *Scope) FieldByName(name string) (field *Field, ok bool) {
	for _, field := range scope.Fields() {
		if field.Name == name || field.DBName == name {
			return field, true
		}
	}
	return nil, false
}

// Raw set sql
func (scope *Scope) Raw(sql string) *Scope {
	scope.Sql = strings.Replace(sql, "$$", "?", -1)
	return scope
}

// Exec invoke sql
func (scope *Scope) Exec() *Scope {
	defer scope.trace(NowFunc())

	if !scope.HasError() {
		if result, err := scope.SqlDB().Exec(scope.Sql, scope.SqlVars...); scope.Err(err) == nil {
			if count, err := result.RowsAffected(); scope.Err(err) == nil {
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

// InstanceID get InstanceID for scope
func (scope *Scope) InstanceID() string {
	if scope.instanceID == "" {
		scope.instanceID = fmt.Sprintf("%v%v", &scope, &scope.db)
	}
	return scope.instanceID
}

func (scope *Scope) InstanceSet(name string, value interface{}) *Scope {
	return scope.Set(name+scope.InstanceID(), value)
}

func (scope *Scope) InstanceGet(name string) (interface{}, bool) {
	return scope.Get(name + scope.InstanceID())
}

// Begin start a transaction
func (scope *Scope) Begin() *Scope {
	if db, ok := scope.SqlDB().(sqlDb); ok {
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
				scope.Err(db.Commit())
			}
			scope.db.db = scope.db.parent.db
		}
	}
	return scope
}

func (scope *Scope) SelectAttrs() []string {
	if scope.selectAttrs == nil {
		attrs := []string{}
		for _, value := range scope.Search.selects {
			if str, ok := value.(string); ok {
				attrs = append(attrs, str)
			} else if strs, ok := value.([]string); ok {
				attrs = append(attrs, strs...)
			} else if strs, ok := value.([]interface{}); ok {
				for _, str := range strs {
					attrs = append(attrs, fmt.Sprintf("%v", str))
				}
			}
		}
		scope.selectAttrs = &attrs
	}
	return *scope.selectAttrs
}

func (scope *Scope) OmitAttrs() []string {
	return scope.Search.omits
}

func (scope *Scope) scan(rows *sql.Rows, columns []string, fields map[string]*Field) {
	var values = make([]interface{}, len(columns))
	var ignored interface{}

	for index, column := range columns {
		if field, ok := fields[column]; ok {
			if field.Field.Kind() == reflect.Ptr {
				values[index] = field.Field.Addr().Interface()
			} else {
				reflectValue := reflect.New(reflect.PtrTo(field.Struct.Type))
				reflectValue.Elem().Set(field.Field.Addr())
				values[index] = reflectValue.Interface()
			}
		} else {
			values[index] = &ignored
		}
	}

	scope.Err(rows.Scan(values...))

	for index, column := range columns {
		if field, ok := fields[column]; ok {
			if field.Field.Kind() != reflect.Ptr {
				if v := reflect.ValueOf(values[index]).Elem().Elem(); v.IsValid() {
					field.Field.Set(v)
				}
			}
		}
	}
}
