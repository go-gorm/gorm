package gorm

import (
	"regexp"

	"reflect"
)

type Model struct {
	Data interface{}
}

func toModel(value interface{}) *Model {
	var model Model
	model.Data = value
	return &model
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

func (m *Model) Columns() (columns []string) {
	typ := reflect.TypeOf(m.Data).Elem()

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if !p.Anonymous {
			columns = append(columns, toSnake(p.Name))
		}
	}

	return
}

func (model *Model) MissingColumns() (results []string) {
	return
}

func (model *Model) ColumnType(column string) (result string) {
	return
}
