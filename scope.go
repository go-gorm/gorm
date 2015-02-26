package gorm

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"reflect"
)

type Scope struct {
	Value           interface{}
	indirectValue   *reflect.Value
	Search          *search
	Sql             string
	SqlVars         []interface{}
	db              *DB
	skipLeft        bool
	primaryKeyField *Field
	instanceId      string
	fields          map[string]*Field
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

func (scope *Scope) PrimaryKeyField() *Field {
	if field := scope.GetModelStruct().PrimaryKeyField; field != nil {
		return scope.Fields()[field.DBName]
	}
	return nil
}

// PrimaryKey get the primary key's column name
func (scope *Scope) PrimaryKey() string {
	if field := scope.PrimaryKeyField(); field != nil {
		return field.DBName
	}
	return ""
}

// PrimaryKeyZero check the primary key is blank or not
func (scope *Scope) PrimaryKeyZero() bool {
	field := scope.PrimaryKeyField()
	return field == nil || field.IsBlank
}

// PrimaryKeyValue get the primary key's value
func (scope *Scope) PrimaryKeyValue() interface{} {
	if field := scope.PrimaryKeyField(); field != nil && field.Field.IsValid() {
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
	} else if dbName, ok := column.(string); ok {
		if field, ok := scope.Fields()[dbName]; ok {
			return field.Set(value)
		}

		dbName = ToDBName(dbName)
		if field, ok := scope.Fields()[dbName]; ok {
			return field.Set(value)
		}
	}
	return errors.New("could not convert column to field")
}

func (scope *Scope) CallMethod(name string, checkError bool) {
	if scope.Value == nil && (!checkError || !scope.HasError()) {
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
				f(scope.NewDB())
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
			call(values.Index(i).Addr().Interface())
		}
	} else {
		call(scope.Value)
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

// TableName get table name
func (scope *Scope) TableName() string {
	if scope.Search != nil && len(scope.Search.TableName) > 0 {
		return scope.Search.TableName
	}
	return scope.GetModelStruct().TableName
}

func (scope *Scope) QuotedTableName() (name string) {
	if scope.Search != nil && len(scope.Search.TableName) > 0 {
		return scope.Quote(scope.Search.TableName)
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
		if field.Name == name {
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
