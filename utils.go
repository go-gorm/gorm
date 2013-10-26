package gorm

import (
	"fmt"
	"reflect"
	"strings"
)

func modelValues(m interface{}) (columns []string, values []interface{}) {
	typ := reflect.TypeOf(m).Elem()

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if !p.Anonymous {
			columns = append(columns, strings.ToLower(p.Name))
			value := reflect.ValueOf(m).Elem().FieldByName(p.Name)
			values = append(values, value.Interface())
		}
	}
	return
}

func valuesToBinVar(values []interface{}) string {
	var sqls []string
	for index, _ := range values {
		sqls = append(sqls, fmt.Sprintf("$%d", index+1))
	}
	return strings.Join(sqls, ",")
}

func quoteMap(values []string) (results []string) {
	for _, value := range values {
		results = append(results, "\""+value+"\"")
	}
	return
}
