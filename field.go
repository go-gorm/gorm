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
	Name           string
	Value          interface{}
	DbName         string
	AutoCreateTime bool
	AutoUpdateTime bool
	IsPrimaryKey   bool
	IsBlank        bool
	structField    reflect.StructField

	beforeAssociation bool
	afterAssociation  bool
	foreignKey        string
	model             *Model
}

func (f *Field) SqlType() string {
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
