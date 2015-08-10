package gorm

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"reflect"
)

type Scope struct {
	Search          *search
	Value           interface{}
	Sql             string
	SqlVars         []interface{}
	db              *DB
	indirectValue   *reflect.Value
	instanceId      string
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

func (scope *Scope) NeedPtr() *Scope {
	reflectKind := reflect.ValueOf(scope.Value).Kind()
	if !((reflectKind == reflect.Invalid) || (reflectKind == reflect.Ptr)) {
		err := fmt.Errorf("%v %v\n", fileWithLineNum(), "using unaddressable value")
		scope.Err(err)
		fmt.Printf(err.Error())
	}
	return scope
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
	} else {
		return scope.Dialect().Quote(str)
	}
}

func (scope *Scope) QuoteIfPossible(str string) string {
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

func (scope *Scope) CallMethod(name string, checkError bool) {
	if scope.Value == nil || (checkError && scope.HasError()) {
		return
	}

	call := func(value interface{}) {
		if fm := reflect.ValueOf(value).MethodByName(name); fm.IsValid() {
			switch f := fm.Interface().(type) {
			case func():
				f()
			case func(s *Scope):
				f(scope)
			case func(s *DB):
				newDB := scope.NewDB()
				f(newDB)
				scope.Err(newDB.Error)
			case func() error:
				scope.Err(f())
			case func(s *Scope) error:
				scope.Err(f(scope))
			case func(s *DB) error:
				scope.Err(f(scope.NewDB()))
			default:
				scope.Err(fmt.Errorf("unsupported function %v", name))
			}
		}
	}

	if values := scope.IndirectValue(); values.Kind() == reflect.Slice {
		for i := 0; i < values.Len(); i++ {
			value := values.Index(i).Addr().Interface()
			if values.Index(i).Kind() == reflect.Ptr {
				value = values.Index(i).Interface()
			}
			call(value)
		}
	} else {
		if scope.IndirectValue().CanAddr() {
			call(scope.IndirectValue().Addr().Interface())
		} else {
			call(scope.IndirectValue().Interface())
		}
	}
}

func (scope *Scope) CallMethodWithErrorCheck(name string) {
	scope.CallMethod(name, true)
}

// AddToVars add value as sql's vars, gorm will escape them
func (scope *Scope) AddToVars(value interface{}) string {
	if expr, ok := value.(*expr); ok {
		exp := expr.expr
		for _, arg := range expr.args {
			exp = strings.Replace(exp, "?", scope.AddToVars(arg), 1)
		}
		return exp
	} else {
		scope.SqlVars = append(scope.SqlVars, value)
		return scope.Dialect().BinVar(len(scope.SqlVars))
	}
}

type tabler interface {
	TableName() string
}

type dbTabler interface {
	TableName(*DB) string
}

// TableName get table name
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

func (scope *Scope) QuotedTableName() (name string) {
	if scope.Search != nil && len(scope.Search.tableName) > 0 {
		if strings.Index(scope.Search.tableName, " ") != -1 {
			return scope.Search.tableName
		}
		return scope.Quote(scope.Search.tableName)
	} else {
		return scope.Quote(scope.TableName())
	}
}

// CombinedConditionSql get combined condition sql
func (scope *Scope) CombinedConditionSql() string {
	return scope.joinsSql() + scope.whereSql() + scope.groupSql() +
		scope.havingSql() + scope.orderSql() + scope.limitSql() + scope.offsetSql()
}

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
	defer scope.Trace(NowFunc())

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

// InstanceId get InstanceId for scope
func (scope *Scope) InstanceId() string {
	if scope.instanceId == "" {
		scope.instanceId = fmt.Sprintf("%v%v", &scope, &scope.db)
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
				db.Commit()
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

func (scope *Scope) changeableDBColumn(column string) bool {
	selectAttrs := scope.SelectAttrs()
	omitAttrs := scope.OmitAttrs()

	if len(selectAttrs) > 0 {
		for _, attr := range selectAttrs {
			if column == ToDBName(attr) {
				return true
			}
		}
		return false
	}

	for _, attr := range omitAttrs {
		if column == ToDBName(attr) {
			return false
		}
	}
	return true
}

func (scope *Scope) changeableField(field *Field) bool {
	selectAttrs := scope.SelectAttrs()
	omitAttrs := scope.OmitAttrs()

	if len(selectAttrs) > 0 {
		for _, attr := range selectAttrs {
			if field.Name == attr || field.DBName == attr {
				return true
			}
		}
		return false
	}

	for _, attr := range omitAttrs {
		if field.Name == attr || field.DBName == attr {
			return false
		}
	}

	return !field.IsIgnored
}

func (scope *Scope) shouldSaveAssociations() bool {
	saveAssociations, ok := scope.Get("gorm:save_associations")
	if ok && !saveAssociations.(bool) {
		return false
	}
	return true
}
