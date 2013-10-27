package gorm

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type Model struct {
	Data   interface{}
	driver string
}

type Field struct {
	Name           string
	Value          interface{}
	SqlType        string
	DbName         string
	AutoCreateTime bool
	AutoUpdateTime bool
	IsPrimaryKey   bool
}

func (s *Orm) toModel(value interface{}) *Model {
	return &Model{Data: value, driver: s.driver}
}

func (m *Model) PrimaryKeyIsEmpty() bool {
	return m.PrimaryKeyValue() == 0
}

func (m *Model) PrimaryKeyValue() int64 {
	t := reflect.TypeOf(m.Data).Elem()
	switch t.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice:
		return 0
	default:
		result := reflect.ValueOf(m.Data).Elem()
		value := result.FieldByName(m.PrimaryKey())
		if value.IsValid() {
			return result.FieldByName(m.PrimaryKey()).Interface().(int64)
		} else {
			return 0
		}
	}
}

func (m *Model) PrimaryKey() string {
	return "Id"
}

func (m *Model) PrimaryKeyDb() string {
	return toSnake(m.PrimaryKey())
}

func (m *Model) Fields(operation string) (fields []Field) {
	typ := reflect.TypeOf(m.Data).Elem()

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if !p.Anonymous {
			var field Field
			field.Name = p.Name
			field.DbName = toSnake(p.Name)
			field.IsPrimaryKey = m.PrimaryKeyDb() == field.DbName
			field.AutoCreateTime = "created_at" == field.DbName
			field.AutoUpdateTime = "updated_at" == field.DbName
			value := reflect.ValueOf(m.Data).Elem().FieldByName(p.Name)

			switch operation {
			case "create":
				if (field.AutoCreateTime || field.AutoUpdateTime) && value.Interface().(time.Time).IsZero() {
					value = reflect.ValueOf(time.Now())
					reflect.ValueOf(m.Data).Elem().FieldByName(p.Name).Set(value)
				}
			case "update":
				if field.AutoUpdateTime {
					value = reflect.ValueOf(time.Now())
					reflect.ValueOf(m.Data).Elem().FieldByName(p.Name).Set(value)
				}
			default:
			}
			field.Value = value.Interface()

			if field.IsPrimaryKey {
				field.SqlType = getPrimaryKeySqlType(m.driver, field.Value, 0)
			} else {
				field.SqlType = getSqlType(m.driver, field.Value, 0)
			}
			fields = append(fields, field)
		}
	}
	return
}

func (m *Model) ColumnsAndValues(operation string) (columns []string, values []interface{}) {
	for _, field := range m.Fields(operation) {
		if !field.IsPrimaryKey {
			columns = append(columns, field.DbName)
			values = append(values, field.Value)
		}
	}
	return
}

func (m *Model) TableName() string {
	t := reflect.TypeOf(m.Data)
	for {
		c := false
		switch t.Kind() {
		case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice:
			t = t.Elem()
			c = true
		}
		if !c {
			break
		}
	}
	reg, _ := regexp.Compile("s*$")
	return reg.ReplaceAllString(toSnake(t.Name()), "s")
}

func (m *Model) callMethod(method string) error {
	fm := reflect.ValueOf(m.Data).MethodByName(method)
	if fm.IsValid() {
		v := fm.Call([]reflect.Value{})
		if len(v) > 0 {
			if verr, ok := v[0].Interface().(error); ok {
				return verr
			}
		}
	}
	return nil
}

func (model *Model) MissingColumns() (results []string) {
	return
}

func (model *Model) CreateTable() (sql string) {
	var sqls []string
	for _, field := range model.Fields("null") {
		sqls = append(sqls, field.DbName+" "+field.SqlType)
	}

	sql = fmt.Sprintf(
		"CREATE TABLE \"%v\" (%v)",
		model.TableName(),
		strings.Join(sqls, ","),
	)
	return
}

func (model *Model) ReturningStr() (str string) {
	if model.driver == "postgres" {
		str = fmt.Sprintf("RETURNING \"%v\"", model.PrimaryKeyDb())
	}
	return
}
