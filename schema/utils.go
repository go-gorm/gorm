package schema

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/jinzhu/gorm/utils"
)

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

func removeSettingFromTag(tag reflect.StructTag, name string) reflect.StructTag {
	return reflect.StructTag(regexp.MustCompile(`(?i)(gorm:.*?)(`+name+`:.*?)(;|("))`).ReplaceAllString(string(tag), "${1}${4}"))
}

// GetRelationsValues get relations's values from a reflect value
func GetRelationsValues(reflectValue reflect.Value, rels []*Relationship) (reflectResults reflect.Value) {
	for _, rel := range rels {
		reflectResults = reflect.MakeSlice(reflect.SliceOf(rel.FieldSchema.ModelType), 0, 0)

		appendToResults := func(value reflect.Value) {
			if _, isZero := rel.Field.ValueOf(value); !isZero {
				result := reflect.Indirect(rel.Field.ReflectValueOf(value))
				switch result.Kind() {
				case reflect.Struct:
					reflectResults = reflect.Append(reflectResults, result)
				case reflect.Slice, reflect.Array:
					for i := 0; i < value.Len(); i++ {
						reflectResults = reflect.Append(reflectResults, reflect.Indirect(value.Index(i)))
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
		fieldValues = make([]reflect.Value, len(fields))
		results     = [][]interface{}{}
		dataResults = map[string][]reflect.Value{}
	)

	switch reflectValue.Kind() {
	case reflect.Struct:
		results = [][]interface{}{make([]interface{}, len(fields))}

		for idx, field := range fields {
			fieldValues[idx] = field.ReflectValueOf(reflectValue)
			results[0][idx] = fieldValues[idx].Interface()
		}

		dataResults[utils.ToStringKey(fieldValues...)] = []reflect.Value{reflectValue}
	case reflect.Slice, reflect.Array:
		for i := 0; i < reflectValue.Len(); i++ {
			for idx, field := range fields {
				fieldValues[idx] = field.ReflectValueOf(reflectValue.Index(i))
			}

			dataKey := utils.ToStringKey(fieldValues...)
			if _, ok := dataResults[dataKey]; !ok {
				result := make([]interface{}, len(fieldValues))
				for idx, fieldValue := range fieldValues {
					result[idx] = fieldValue.Interface()
				}
				results = append(results, result)

				dataResults[dataKey] = []reflect.Value{reflectValue.Index(i)}
			} else {
				dataResults[dataKey] = append(dataResults[dataKey], reflectValue.Index(i))
			}
		}
	}

	return dataResults, results
}

// ToQueryValues to query values
func ToQueryValues(foreignKeys []string, foreignValues [][]interface{}) (interface{}, []interface{}) {
	queryValues := make([]interface{}, len(foreignValues))
	if len(foreignKeys) == 1 {
		for idx, r := range foreignValues {
			queryValues[idx] = r[0]
		}

		return foreignKeys[0], queryValues
	} else {
		for idx, r := range foreignValues {
			queryValues[idx] = r
		}
	}
	return foreignKeys, queryValues
}
