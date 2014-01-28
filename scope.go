package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm/dialect"
	"go/ast"
	"strconv"
	"strings"
	"time"

	"reflect"
	"regexp"
)

type Scope struct {
	Value    interface{}
	Search   *search
	Sql      string
	SqlVars  []interface{}
	db       *DB
	_values  map[string]interface{}
	skipLeft bool
}

func (scope *Scope) Quote(str string) string {
	return scope.Dialect().Quote(str)
}

func (db *DB) NewScope(value interface{}) *Scope {
	db.Value = value
	return &Scope{db: db, Search: db.search, Value: value, _values: map[string]interface{}{}}
}

func (scope *Scope) SkipLeft() {
	scope.skipLeft = true
}

func (scope *Scope) callCallbacks(funcs []*func(s *Scope)) *Scope {
	for _, f := range funcs {
		(*f)(scope)
		if scope.skipLeft {
			break
		}
	}
	return scope
}

func (scope *Scope) New(value interface{}) *Scope {
	return &Scope{db: scope.db.parent, Search: &search{}, Value: value}
}

func (scope *Scope) NewDB() *DB {
	return scope.db.new()
}

func (scope *Scope) DB() sqlCommon {
	return scope.db.db
}

func (scope *Scope) Dialect() dialect.Dialect {
	return scope.db.parent.dialect
}

func (scope *Scope) Err(err error) error {
	if err != nil {
		scope.db.err(err)
	}
	return err
}

func (scope *Scope) Log(v ...interface{}) {
	scope.db.log(v...)
}

func (scope *Scope) HasError() bool {
	return scope.db.Error != nil
}

func (scope *Scope) PrimaryKey() string {
	return "id"
}

func (scope *Scope) PrimaryKeyZero() bool {
	return isBlank(reflect.ValueOf(scope.PrimaryKeyValue()))
}

func (scope *Scope) PrimaryKeyValue() interface{} {
	data := reflect.Indirect(reflect.ValueOf(scope.Value))

	if data.Kind() == reflect.Struct {
		if field := data.FieldByName(snakeToUpperCamel(scope.PrimaryKey())); field.IsValid() {
			return field.Interface()
		}
	}
	return 0
}

func (scope *Scope) HasColumn(name string) bool {
	_, result := scope.FieldByName(name)
	return result
}

func (scope *Scope) FieldByName(name string) (interface{}, bool) {
	data := reflect.Indirect(reflect.ValueOf(scope.Value))

	if data.Kind() == reflect.Struct {
		if field := data.FieldByName(name); field.IsValid() {
			return field.Interface(), true
		}
	} else if data.Kind() == reflect.Slice {
		return nil, reflect.New(data.Type().Elem()).Elem().FieldByName(name).IsValid()
	}
	return nil, false
}

func (scope *Scope) updatedAttrsWithValues(values map[string]interface{}, ignoreProtectedAttrs bool) (results map[string]interface{}, hasUpdate bool) {
	data := reflect.Indirect(reflect.ValueOf(scope.Value))
	if !data.CanAddr() {
		return values, true
	}

	for key, value := range values {
		if field := data.FieldByName(snakeToUpperCamel(key)); field.IsValid() {
			if field.Interface() != value {

				switch field.Kind() {
				case reflect.Int, reflect.Int32, reflect.Int64:
					if s, ok := value.(string); ok {
						i, err := strconv.Atoi(s)
						if scope.Err(err) == nil {
							value = i
						}
					}

					scope.db.log(field.Int() != reflect.ValueOf(value).Int())
					if field.Int() != reflect.ValueOf(value).Int() {
						hasUpdate = true
						setFieldValue(field, value)
					}
				default:
					hasUpdate = true
					setFieldValue(field, value)
				}
			}
		}
	}
	return
}

func (scope *Scope) SetColumn(column string, value interface{}) {
	if scope.Value == nil {
		return
	}

	data := reflect.Indirect(reflect.ValueOf(scope.Value))
	setFieldValue(data.FieldByName(snakeToUpperCamel(column)), value)
}

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

	if values := reflect.Indirect(reflect.ValueOf(scope.Value)); values.Kind() == reflect.Slice {
		for i := 0; i < values.Len(); i++ {
			call(values.Index(i).Addr().Interface())
		}
	} else {
		call(scope.Value)
	}
}

func (scope *Scope) AddToVars(value interface{}) string {
	scope.SqlVars = append(scope.SqlVars, value)
	return scope.Dialect().BinVar(len(scope.SqlVars))
}

func (scope *Scope) TableName() string {
	if scope.Search != nil && len(scope.Search.TableName) > 0 {
		return scope.Search.TableName
	} else {
		if scope.Value == nil {
			scope.Err(errors.New("can't get table name"))
			return ""
		}
		data := reflect.Indirect(reflect.ValueOf(scope.Value))

		if data.Kind() == reflect.Slice {
			data = reflect.New(data.Type().Elem()).Elem()
		}

		if fm := data.MethodByName("TableName"); fm.IsValid() {
			if v := fm.Call([]reflect.Value{}); len(v) > 0 {
				if result, ok := v[0].Interface().(string); ok {
					return result
				}
			}
		}

		str := toSnake(data.Type().Name())

		if !scope.db.parent.singularTable {
			pluralMap := map[string]string{"ch": "ches", "ss": "sses", "sh": "shes", "day": "days", "y": "ies", "x": "xes", "s?": "s"}
			for key, value := range pluralMap {
				reg := regexp.MustCompile(key + "$")
				if reg.MatchString(str) {
					return reg.ReplaceAllString(str, value)
				}
			}
		}

		return str
	}
}

func (s *Scope) CombinedConditionSql() string {
	return s.joinsSql() + s.whereSql() + s.groupSql() + s.havingSql() + s.orderSql() + s.limitSql() + s.offsetSql()
}

func (scope *Scope) SqlTagForField(field *Field) (tag string) {
	value := field.Value
	reflect_value := reflect.ValueOf(value)

	if field.IsScanner() {
		value = reflect_value.Field(0).Interface()
	}

	switch reflect_value.Kind() {
	case reflect.Slice:
		if _, ok := value.([]byte); !ok {
			return
		}
	case reflect.Struct:
		if !field.IsTime() && !field.IsScanner() {
			return
		}
	}

	tag = field.Tag
	if len(tag) == 0 && tag != "-" {
		if field.isPrimaryKey {
			tag = scope.Dialect().PrimaryKeyTag(value, field.Size)
		} else {
			tag = scope.Dialect().SqlTag(value, field.Size)
		}
	}

	if len(field.AddationalTag) > 0 {
		tag = tag + " " + field.AddationalTag
	}
	return
}

func (scope *Scope) Fields() []*Field {
	indirectValue := reflect.Indirect(reflect.ValueOf(scope.Value))
	fields := []*Field{}

	if !indirectValue.IsValid() {
		return fields
	}

	scopeTyp := indirectValue.Type()
	for i := 0; i < scopeTyp.NumField(); i++ {
		fieldStruct := scopeTyp.Field(i)
		if fieldStruct.Anonymous || !ast.IsExported(fieldStruct.Name) {
			continue
		}

		var field Field
		field.Name = fieldStruct.Name
		field.DBName = toSnake(fieldStruct.Name)

		value := indirectValue.FieldByName(fieldStruct.Name)
		field.Value = value.Interface()
		field.IsBlank = isBlank(value)

		if scope.db != nil {
			tag, addationalTag, size := parseSqlTag(fieldStruct.Tag.Get(scope.db.parent.tagIdentifier))
			field.Tag = tag
			field.AddationalTag = addationalTag
			field.isPrimaryKey = scope.PrimaryKey() == field.DBName
			field.Size = size

			field.SqlTag = scope.SqlTagForField(&field)

			if tag == "-" {
				field.IsIgnored = true
			}

			// parse association
			elem := reflect.Indirect(value)
			typ := elem.Type()

			switch elem.Kind() {
			case reflect.Slice:
				typ = typ.Elem()

				if _, ok := field.Value.([]byte); !ok {
					foreignKey := scopeTyp.Name() + "Id"
					if reflect.New(typ).Elem().FieldByName(foreignKey).IsValid() {
						field.ForeignKey = foreignKey
					}
					field.AfterAssociation = true
				}
			case reflect.Struct:
				if !field.IsTime() && !field.IsScanner() {
					if scope.HasColumn(field.Name + "Id") {
						field.ForeignKey = field.Name + "Id"
						field.BeforeAssociation = true
					} else {
						foreignKey := scopeTyp.Name() + "Id"
						if reflect.New(typ).Elem().FieldByName(foreignKey).IsValid() {
							field.ForeignKey = foreignKey
						}
						field.AfterAssociation = true
					}
				}
			}
		}
		fields = append(fields, &field)
	}

	return fields
}

func (scope *Scope) Raw(sql string) *Scope {
	scope.Sql = strings.Replace(sql, "$$", "?", -1)
	return scope
}

func (scope *Scope) Exec() *Scope {
	defer scope.Trace(time.Now())

	if !scope.HasError() {
		_, err := scope.DB().Exec(scope.Sql, scope.SqlVars...)
		scope.Err(err)
	}
	return scope
}

func (scope *Scope) Get(name string) (value interface{}, ok bool) {
	value, ok = scope._values[name]
	return
}

func (scope *Scope) Set(name string, value interface{}) *Scope {
	scope._values[name] = value
	return scope
}

func (scope *Scope) Trace(t time.Time) {
	if len(scope.Sql) > 0 {
		scope.db.slog(scope.Sql, t, scope.SqlVars...)
	}
}

func (scope *Scope) Begin() *Scope {
	if db, ok := scope.DB().(sqlDb); ok {
		if tx, err := db.Begin(); err == nil {
			scope.db.db = interface{}(tx).(sqlCommon)
			scope.Set("gorm:started_transaction", true)
		}
	}
	return scope
}

func (scope *Scope) CommitOrRollback() *Scope {
	if _, ok := scope.Get("gorm:started_transaction"); ok {
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

func (scope *Scope) row() *sql.Row {
	defer scope.Trace(time.Now())
	scope.prepareQuerySql()
	return scope.DB().QueryRow(scope.Sql, scope.SqlVars...)
}

func (scope *Scope) rows() (*sql.Rows, error) {
	defer scope.Trace(time.Now())
	scope.prepareQuerySql()
	return scope.DB().Query(scope.Sql, scope.SqlVars...)
}

func (scope *Scope) initialize() *Scope {
	for _, clause := range scope.Search.WhereConditions {
		scope.updatedAttrsWithValues(convertInterfaceToMap(clause["query"]), false)
	}
	scope.updatedAttrsWithValues(convertInterfaceToMap(scope.Search.InitAttrs), false)
	scope.updatedAttrsWithValues(convertInterfaceToMap(scope.Search.AssignAttrs), false)
	return scope
}

func (scope *Scope) pluck(column string, value interface{}) *Scope {
	dest := reflect.Indirect(reflect.ValueOf(value))
	scope.Search = scope.Search.clone().selects(column)
	if dest.Kind() != reflect.Slice {
		scope.Err(errors.New("Results should be a slice"))
		return scope
	}

	rows, err := scope.rows()
	if scope.Err(err) == nil {
		defer rows.Close()
		for rows.Next() {
			elem := reflect.New(dest.Type().Elem()).Interface()
			scope.Err(rows.Scan(elem))
			dest.Set(reflect.Append(dest, reflect.ValueOf(elem).Elem()))
		}
	}
	return scope
}

func (scope *Scope) count(value interface{}) *Scope {
	scope.Search = scope.Search.clone().selects("count(*)")
	scope.Err(scope.row().Scan(value))
	return scope
}

func (scope *Scope) typeName() string {
	value := reflect.Indirect(reflect.ValueOf(scope.Value))
	if value.Kind() == reflect.Slice {
		return value.Type().Elem().Name()
	} else {
		return value.Type().Name()
	}
}

func (scope *Scope) related(value interface{}, foreignKeys ...string) *Scope {
	toScope := scope.New(value)

	for _, foreignKey := range append(foreignKeys, toScope.typeName()+"Id", scope.typeName()+"Id") {
		if foreignValue, ok := scope.FieldByName(foreignKey); ok {
			return toScope.inlineCondition(foreignValue).callCallbacks(scope.db.parent.callback.queries)
		} else if toScope.HasColumn(foreignKey) {
			sql := fmt.Sprintf("%v = ?", scope.Quote(toSnake(foreignKey)))
			return toScope.inlineCondition(sql, scope.PrimaryKeyValue()).callCallbacks(scope.db.parent.callback.queries)
		}
	}
	return scope
}
