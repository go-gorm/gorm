package gorm

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type Model struct {
	Data   interface{}
	driver string
}

type Field struct {
	Name    string
	Value   interface{}
	SqlType string
	DbName  string
}

func (s *Orm) toModel(value interface{}) *Model {
	return &Model{Data: value, driver: s.driver}
}

func (m *Model) PrimaryKey() string {
	return "Id"
}

func (m *Model) Fields() (fields []Field) {
	typ := reflect.TypeOf(m.Data).Elem()

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if !p.Anonymous {
			var field Field
			field.Name = p.Name
			field.DbName = toSnake(p.Name)
			field.Value = reflect.ValueOf(m.Data).Elem().FieldByName(p.Name).Interface()
			if m.PrimaryKey() == p.Name {
				field.SqlType = getPrimaryKeySqlType(m.driver, field.Value, 0)
			} else {
				field.SqlType = getSqlType(m.driver, field.Value, 0)
			}
			fields = append(fields, field)
		}
	}
	return
}

func (m *Model) ColumnsAndValues() (columns []string, values []interface{}) {
	typ := reflect.TypeOf(m.Data).Elem()

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if !p.Anonymous {
			columns = append(columns, toSnake(p.Name))
			value := reflect.ValueOf(m.Data).Elem().FieldByName(p.Name)
			values = append(values, value.Interface())
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

func (model *Model) MissingColumns() (results []string) {
	return
}

func (model *Model) ColumnType(column string) (result string) {
	return
}

func (model *Model) CreateTable() (sql string) {
	var sqls []string
	for _, field := range model.Fields() {
		sqls = append(sqls, field.DbName+" "+field.SqlType)
	}

	sql = fmt.Sprintf(
		"CREATE TABLE \"%v\" (%v)",
		model.TableName(),
		strings.Join(sqls, ","),
	)
	return
}
