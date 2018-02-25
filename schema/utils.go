package schema

import (
	"bytes"
	"reflect"
	"strings"
)

type strCase bool

const (
	lower strCase = false
	upper strCase = true
)

// ToDBName convert str to db name
func ToDBName(name string) string {
	if name == "" {
		return ""
	}
	var (
		value                        = name
		buf                          = bytes.NewBufferString("")
		lastCase, currCase, nextCase strCase
	)

	for i, v := range value[:len(value)-1] {
		nextCase = strCase(value[i+1] >= 'A' && value[i+1] <= 'Z')
		if i > 0 {
			if currCase == upper {
				if lastCase == upper && nextCase == upper {
					buf.WriteRune(v)
				} else {
					if value[i-1] != '_' && value[i+1] != '_' {
						buf.WriteRune('_')
					}
					buf.WriteRune(v)
				}
			} else {
				buf.WriteRune(v)
				if i == len(value)-2 && nextCase == upper {
					buf.WriteRune('_')
				}
			}
		} else {
			currCase = upper
			buf.WriteRune(v)
		}
		lastCase = currCase
		currCase = nextCase
	}

	buf.WriteByte(value[len(value)-1])
	return strings.ToLower(buf.String())
}

func checkTruth(val string) bool {
	if strings.ToLower(val) == "false" {
		return false
	}
	return true
}

func toColumns(val string) []string {
	return strings.Split(val, ",")
}

func getSchemaField(name string, fields []*Field) *Field {
	for _, field := range fields {
		if field.Name == name || field.DBName == name {
			return field
		}
	}
	return nil
}

func getPrimaryPrimaryField(fields []*Field) *Field {
	for _, field := range fields {
		if field.DBName == "id" {
			return field
		}
	}
	return fields[0]
}

func parseTagSetting(tags reflect.StructTag) map[string]string {
	setting := map[string]string{}
	for _, str := range []string{tags.Get("sql"), tags.Get("gorm")} {
		if str != "" {
			tags := strings.Split(str, ";")
			for _, value := range tags {
				v := strings.Split(value, ":")
				k := strings.TrimSpace(strings.ToUpper(v[0]))
				if len(v) >= 2 {
					setting[k] = strings.Join(v[1:], ":")
				} else {
					setting[k] = k
				}
			}
		}
	}
	return setting
}
