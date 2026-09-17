package schema

import (
	"context"
	"fmt"
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

	var parsedNames []string
	for i := 0; i < len(names); i++ {
		s := names[i]
		for strings.HasSuffix(s, "\\") && i+1 < len(names) {
			i++
			s = s[:len(s)-1] + sep + names[i]
		}
		parsedNames = append(parsedNames, s)
	}

	for _, tag := range parsedNames {
		values := strings.Split(tag, ":")
		k := strings.TrimSpace(strings.ToUpper(values[0]))
		if len(values) >= 2 {
			val := strings.Join(values[1:], ":")
			val = strings.ReplaceAll(val, `\"`, `"`)
			settings[k] = val

			// `<dialector>:type:<value>` declares a database-specific data type, e.g.
			// `mysql:type:json;postgres:type:text[]`, which is resolved with the
			// dialector name when parsing the schema (see dialectTypeSetting).
			// The plain setting above is kept for backward compatibility with
			// three-segment tags like `index:type:fulltext`.
			if len(values) >= 3 && strings.TrimSpace(strings.ToUpper(values[1])) == "TYPE" {
				dialectVal := strings.Join(values[2:], ":")
				settings[k+":TYPE"] = strings.ReplaceAll(dialectVal, `\"`, `"`)
			}
		} else if k != "" {
			settings[k] = k
		}
	}

	return settings
}

// dialectTypeAliases maps dialector names to alternative names that are also
// accepted in database-specific type tags, e.g. `postgresql:type:text[]` is
// applied when the dialector name is `postgres`. The exact dialector name is
// always tried first, so `postgres:type:...` and `gaussdb:type:...` can coexist
// on the same field.
var dialectTypeAliases = map[string][]string{
	"mysql":       {"mariadb"},
	"mariadb":     {"mysql"},
	"tidb":        {"mysql"},
	"postgres":    {"postgresql", "pg"},
	"postgresql":  {"postgres", "pg"},
	"cockroachdb": {"postgres", "postgresql"},
	"sqlite":      {"sqlite3"},
	"sqlserver":   {"mssql"},
	"mssql":       {"sqlserver"},
	"gaussdb":     {"opengauss", "postgres", "postgresql"},
	"opengauss":   {"gaussdb", "postgres", "postgresql"},
}

// dialectTypeSetting looks up the database-specific type tag (`<dialect>:type:<value>`,
// stored as `<DIALECT>:TYPE` by ParseTagSetting) for the given dialector name,
// trying the exact name first and then its well-known aliases. It reports false
// when no non-empty setting matches, so the generic `type` tag or GORM's default
// data type rules apply instead.
func dialectTypeSetting(dialectName string, settings map[string]string) (string, bool) {
	dialectName = strings.ToLower(strings.TrimSpace(dialectName))
	if dialectName == "" {
		return "", false
	}

	names := append([]string{dialectName}, dialectTypeAliases[dialectName]...)
	for _, name := range names {
		if v, ok := settings[strings.ToUpper(name)+":TYPE"]; ok && v != "" {
			return v, true
		}
	}
	return "", false
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

func appendSettingFromTag(tag reflect.StructTag, value string) reflect.StructTag {
	t := tag.Get("gorm")
	if strings.Contains(t, value) {
		return tag
	}
	return reflect.StructTag(fmt.Sprintf(`gorm:"%s;%s"`, value, t))
}

// GetRelationsValues get relations's values from a reflect value
func GetRelationsValues(ctx context.Context, reflectValue reflect.Value, rels []*Relationship) (reflectResults reflect.Value) {
	for _, rel := range rels {
		reflectResults = reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(rel.FieldSchema.ModelType)), 0, 1)

		appendToResults := func(value reflect.Value) {
			if _, isZero := rel.Field.ValueOf(ctx, value); !isZero {
				result := reflect.Indirect(rel.Field.ReflectValueOf(ctx, value))
				switch result.Kind() {
				case reflect.Struct:
					reflectResults = reflect.Append(reflectResults, result.Addr())
				case reflect.Slice, reflect.Array:
					for i := 0; i < result.Len(); i++ {
						if elem := result.Index(i); elem.Kind() == reflect.Ptr {
							reflectResults = reflect.Append(reflectResults, elem)
						} else {
							reflectResults = reflect.Append(reflectResults, elem.Addr())
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
func GetIdentityFieldValuesMap(ctx context.Context, reflectValue reflect.Value, fields []*Field) (map[string][]reflect.Value, [][]interface{}) {
	var (
		results       = [][]interface{}{}
		dataResults   = map[string][]reflect.Value{}
		loaded        = map[interface{}]bool{}
		notZero, zero bool
	)

	if reflectValue.Kind() == reflect.Ptr ||
		reflectValue.Kind() == reflect.Interface {
		reflectValue = reflectValue.Elem()
	}

	switch reflectValue.Kind() {
	case reflect.Map:
		results = [][]interface{}{make([]interface{}, len(fields))}
		for idx, field := range fields {
			mapValue := reflectValue.MapIndex(reflect.ValueOf(field.DBName))
			if mapValue.IsZero() {
				mapValue = reflectValue.MapIndex(reflect.ValueOf(field.Name))
			}
			results[0][idx] = mapValue.Interface()
		}

		dataResults[utils.ToStringKey(results[0]...)] = []reflect.Value{reflectValue}
	case reflect.Struct:
		results = [][]interface{}{make([]interface{}, len(fields))}

		for idx, field := range fields {
			results[0][idx], zero = field.ValueOf(ctx, reflectValue)
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
			if elem.Kind() != reflect.Ptr && elem.CanAddr() {
				elemKey = elem.Addr().Interface()
			}

			if _, ok := loaded[elemKey]; ok {
				continue
			}
			loaded[elemKey] = true

			fieldValues := make([]interface{}, len(fields))
			notZero = false
			for idx, field := range fields {
				fieldValues[idx], zero = field.ValueOf(ctx, elem)
				notZero = notZero || !zero
			}

			if notZero {
				dataKey := utils.ToStringKey(fieldValues...)
				if _, ok := dataResults[dataKey]; !ok {
					results = append(results, fieldValues)
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
func GetIdentityFieldValuesMapFromValues(ctx context.Context, values []interface{}, fields []*Field) (map[string][]reflect.Value, [][]interface{}) {
	resultsMap := map[string][]reflect.Value{}
	results := [][]interface{}{}

	for _, v := range values {
		rm, rs := GetIdentityFieldValuesMap(ctx, reflect.Indirect(reflect.ValueOf(v)), fields)
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
	}

	columns := make([]clause.Column, len(foreignKeys))
	for idx, key := range foreignKeys {
		columns[idx] = clause.Column{Table: table, Name: key}
	}

	for idx, r := range foreignValues {
		queryValues[idx] = r
	}

	return columns, queryValues
}

type embeddedNamer struct {
	Table string
	Namer
}
