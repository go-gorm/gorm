package gorm

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"

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

func (m *Model) primaryKeyZero() bool {
	return m.primaryKeyValue() == 0
}

func (m *Model) primaryKeyValue() int64 {
	if m.Data == nil {
		return 0
	}

	t := reflect.TypeOf(m.Data).Elem()
	switch t.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice:
		return 0
	default:
		result := reflect.ValueOf(m.Data).Elem()
		value := result.FieldByName(m.primaryKey())
		if value.IsValid() {
			return result.FieldByName(m.primaryKey()).Interface().(int64)
		} else {
			return 0
		}
	}
}

func (m *Model) primaryKey() string {
	return "Id"
}

func (m *Model) primaryKeyDb() string {
	return toSnake(m.primaryKey())
}

func (m *Model) fields(operation string) (fields []Field) {
	typ := reflect.TypeOf(m.Data).Elem()

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if !p.Anonymous {
			var field Field
			field.Name = p.Name
			field.DbName = toSnake(p.Name)
			field.IsPrimaryKey = m.primaryKeyDb() == field.DbName
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

func (m *Model) columnsAndValues(operation string) (columns []string, values []interface{}) {
	for _, field := range m.fields(operation) {
		if !field.IsPrimaryKey {
			columns = append(columns, field.DbName)
			values = append(values, field.Value)
		}
	}
	return
}

func (m *Model) tableName() (str string, err error) {
	if m.Data == nil {
		err = errors.New("Model haven't been set")
		return
	}

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

	str = toSnake(t.Name())

	pluralMap := map[string]string{"ch": "ches", "ss": "sses", "sh": "shes", "day": "days", "y": "ies", "x": "xes", "s?": "s"}
	for key, value := range pluralMap {
		reg := regexp.MustCompile(key + "$")
		if reg.MatchString(str) {
			return reg.ReplaceAllString(str, value), err
		}
	}
	return
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

func (model *Model) missingColumns() (results []string) {
	return
}

func (model *Model) returningStr() (str string) {
	if model.driver == "postgres" {
		str = fmt.Sprintf("RETURNING \"%v\"", model.primaryKeyDb())
	}
	return
}
