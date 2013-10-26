package gorm

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

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

func toSnake(s string) string {
	buf := bytes.NewBufferString("")
	for i, v := range s {
		if i > 0 && v >= 'A' && v <= 'Z' {
			buf.WriteRune('_')
		}
		buf.WriteRune(v)
	}
	return strings.ToLower(buf.String())
}

func snakeToUpperCamel(s string) string {
	buf := bytes.NewBufferString("")
	for _, v := range strings.Split(s, "_") {
		if len(v) > 0 {
			buf.WriteString(strings.ToUpper(v[:1]))
			buf.WriteString(v[1:])
		}
	}
	return buf.String()
}

func interfaceToTableName(f interface{}) string {
	t := reflect.TypeOf(f)
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

func debug(value interface{}) {
	fmt.Printf("***************\n")
	fmt.Printf("%+v\n\n", value)
}
