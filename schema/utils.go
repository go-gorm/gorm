package schema

import (
	"reflect"
	"regexp"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"
)

var embeddedCacheKey = "embedded_cache_store"

func ParseTagSetting(str string, sep string) map[string]string {
	settings := map[string]string{}
	names := strings.Split(str, sep)

	for i := 0; i < len(names); i++ {
		j := i
		if len(names[j]) > 0 {
			for {
				if names[j][len(names[j])-1] == '\\' {
					i++
					names[j] = names[j][0:len(names[j])-1] + sep + names[i]
					names[i] = ""
				} else {
					break
				}
			}
		}

		values := strings.Split(names[j], ":")
		k := strings.TrimSpace(strings.ToUpper(values[0]))

		if len(values) >= 2 {
			settings[k] = strings.Join(values[1:], ":")
		} else if k != "" {
			settings[k] = k
		}
	}

	return settings
}

func toColumns(val string) (results []string) {
	if val != "" {
		for _, v := range strings.Split(val, ",") {
			results = append(results, strings.TrimSpace(v))
		}
	}
	return
}

func removeSettingFromTag(tag reflect.StructTag, names ...string) reflect.StructTag {
	for _, name := range names {
		tag = reflect.StructTag(regexp.MustCompile(`(?i)(gorm:.*?)(`+name+`(:.*?)?)(;|("))`).ReplaceAllString(string(tag), "${1}${5}"))
	}
	return tag
}

// GetRelationsValues get relations's values from a reflect value
func GetRelationsValues(reflectValue reflect.Value, rels []*Relationship) (reflectResults reflect.Value) {
	for _, rel := range rels {
		reflectResults = reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(rel.FieldSchema.ModelType)), 0, 1)

		appendToResults := func(value reflect.Value) {
			if _, isZero := rel.Field.ValueOf(value); !isZero {
				result := reflect.Indirect(rel.Field.ReflectValueOf(value))
				switch result.Kind() {
				case reflect.Struct:
					reflectResults = reflect.Append(reflectResults, result.Addr())
				case reflect.Slice, reflect.Array:
					for i := 0; i < result.Len(); i++ {
						if result.Index(i).Kind() == reflect.Ptr {
							reflectResults = reflect.Append(reflectResults, result.Index(i))
						} else {
							reflectResults = reflect.Append(reflectResults, result.Index(i).Addr())
						}
					}
				}
			}
		}

		switch reflectValue.Kind() {
		case reflect.Struct:
			appendToResults(reflectValue)
		case reflect.Slice:
			for i := 0; i < reflectValue.Len(); i++ {
				appendToResults(reflectValue.Index(i))
			}
		}

		reflectValue = reflectResults
	}

	return
}

// GetIdentityFieldValuesMap get identity map from fields
func GetIdentityFieldValuesMap(reflectValue reflect.Value, fields []*Field) (map[string][]reflect.Value, [][]interface{}) {
	var (
		results       = [][]interface{}{}
		dataResults   = map[string][]reflect.Value{}
		loaded        = map[interface{}]bool{}
		notZero, zero bool
	)

	switch reflectValue.Kind() {
	case reflect.Struct:
		results = [][]interface{}{make([]interface{}, len(fields))}

		for idx, field := range fields {
			results[0][idx], zero = field.ValueOf(reflectValue)
			notZero = notZero || !zero
		}

		if !notZero {
			return nil, nil
		}

		dataResults[utils.ToStringKey(results[0]...)] = []reflect.Value{reflectValue}
	case reflect.Slice, reflect.Array:
		for i := 0; i < reflectValue.Len(); i++ {
			elem := reflectValue.Index(i)
			elemKey := elem.Interface()
			if elem.Kind() != reflect.Ptr {
				elemKey = elem.Addr().Interface()
			}

			if _, ok := loaded[elemKey]; ok {
				continue
			}
			loaded[elemKey] = true

			fieldValues := make([]interface{}, len(fields))
			notZero = false
			for idx, field := range fields {
				fieldValues[idx], zero = field.ValueOf(elem)
				notZero = notZero || !zero
			}

			if notZero {
				dataKey := utils.ToStringKey(fieldValues...)
				if _, ok := dataResults[dataKey]; !ok {
					results = append(results, fieldValues[:])
					dataResults[dataKey] = []reflect.Value{elem}
				} else {
					dataResults[dataKey] = append(dataResults[dataKey], elem)
				}
			}
		}
	}

	return dataResults, results
}

// GetIdentityFieldValuesMapFromValues get identity map from fields
func GetIdentityFieldValuesMapFromValues(values []interface{}, fields []*Field) (map[string][]reflect.Value, [][]interface{}) {
	resultsMap := map[string][]reflect.Value{}
	results := [][]interface{}{}

	for _, v := range values {
		rm, rs := GetIdentityFieldValuesMap(reflect.Indirect(reflect.ValueOf(v)), fields)
		for k, v := range rm {
			resultsMap[k] = append(resultsMap[k], v...)
		}
		results = append(results, rs...)
	}
	return resultsMap, results
}

// ToQueryValues to query values
func ToQueryValues(table string, foreignKeys []string, foreignValues [][]interface{}) (interface{}, []interface{}) {
	queryValues := make([]interface{}, len(foreignValues))
	if len(foreignKeys) == 1 {
		for idx, r := range foreignValues {
			queryValues[idx] = r[0]
		}

		return clause.Column{Table: table, Name: foreignKeys[0]}, queryValues
	} else {
		columns := make([]clause.Column, len(foreignKeys))
		for idx, key := range foreignKeys {
			columns[idx] = clause.Column{Table: table, Name: key}
		}

		for idx, r := range foreignValues {
			queryValues[idx] = r
		}
		return columns, queryValues
	}
}

type embeddedNamer struct {
	Table string
	Namer
}
