package gorm

import (
	"database/sql"
	"database/sql/driver"

	"time"

	"strconv"
	"strings"

	"reflect"
)

type Field struct {
	Name              string
	Value             interface{}
	DbName            string
	AutoCreateTime    bool
	AutoUpdateTime    bool
	IsPrimaryKey      bool
	structField       reflect.StructField
	modelValue        reflect.Value
	beforeAssociation bool
	afterAssociation  bool
	foreignKey        string
	model             *Model
}

func (f *Field) isBlank() bool {
	value := reflect.ValueOf(f.Value)
	switch value.Kind() {
	case reflect.Int, reflect.Int64, reflect.Int32:
		return value.Int() == 0
	case reflect.String:
		return value.String() == ""
	case reflect.Slice:
		return value.Len() == 0
	case reflect.Struct:
		time_value, is_time := f.Value.(time.Time)
		if is_time {
			return time_value.IsZero()
		} else {
			_, is_scanner := reflect.New(value.Type()).Interface().(sql.Scanner)
			if is_scanner {
				return !value.FieldByName("Valid").Interface().(bool)
			} else {
				m := &Model{data: value.Interface(), do: f.model.do}
				fields := m.columnsHasValue("other")
				if len(fields) == 0 {
					return true
				}
			}
		}
	}
	return false
}

func (f *Field) sqlTag() string {
	column := getInterfaceValue(f.Value)
	field_value := reflect.ValueOf(f.Value)
	switch field_value.Kind() {
	case reflect.Slice:
		return ""
	case reflect.Struct:
		_, is_scanner := reflect.New(field_value.Type()).Interface().(sql.Scanner)
		_, is_time := column.(time.Time)
		if !is_time && !is_scanner {
			return ""
		}
	}

	typ, addational_typ, size := parseSqlTag(f.structField.Tag.Get(tagIdentifier))

	if typ == "-" {
		return ""
	}

	if len(typ) == 0 {
		if f.IsPrimaryKey {
			typ = f.model.do.chain.d.dialect.PrimaryKeyTag(column, size)
		} else {
			typ = f.model.do.chain.d.dialect.SqlTag(column, size)
		}
	}

	if len(addational_typ) > 0 {
		typ = typ + " " + addational_typ
	}
	return typ
}

func (f *Field) parseAssociation() {
	field_value := reflect.ValueOf(f.Value)

	switch field_value.Kind() {
	case reflect.Slice:
		foreign_key := f.model.typeName() + "Id"
		if reflect.New(field_value.Type().Elem()).Elem().FieldByName(foreign_key).IsValid() {
			f.foreignKey = foreign_key
		}
		f.afterAssociation = true
	case reflect.Struct:
		_, is_time := f.Value.(time.Time)
		_, is_scanner := reflect.New(field_value.Type()).Interface().(sql.Scanner)

		if !is_scanner && !is_time {
			if f.modelValue.FieldByName(f.Name + "Id").IsValid() {
				f.foreignKey = f.Name + "Id"
				f.beforeAssociation = true
			} else {
				foreign_key := f.model.typeName() + "Id"
				if reflect.New(field_value.Type()).Elem().FieldByName(foreign_key).IsValid() {
					f.foreignKey = foreign_key
				}
				f.afterAssociation = true
			}
		}
	}
}

func parseSqlTag(str string) (typ string, addational_typ string, size int) {
	if str == "-" {
		typ = str
	} else if str != "" {
		tags := strings.Split(str, ";")
		m := make(map[string]string)
		for _, value := range tags {
			v := strings.Split(value, ":")
			k := strings.Trim(strings.ToUpper(v[0]), " ")
			if len(v) == 2 {
				m[k] = v[1]
			} else {
				m[k] = k
			}
		}

		if len(m["SIZE"]) > 0 {
			size, _ = strconv.Atoi(m["SIZE"])
		}

		if len(m["TYPE"]) > 0 {
			typ = m["TYPE"]
		}

		addational_typ = m["NOT NULL"] + " " + m["UNIQUE"]
	}
	return
}

func getInterfaceValue(column interface{}) interface{} {
	if v, ok := column.(reflect.Value); ok {
		column = v.Interface()
	}

	if valuer, ok := interface{}(column).(driver.Valuer); ok {
		column = reflect.New(reflect.ValueOf(valuer).Field(0).Type()).Elem().Interface()
	}
	return column
}
