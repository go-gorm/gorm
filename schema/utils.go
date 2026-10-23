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

// embeddedCacheKey marks the private cache used while flattening embedded schemas,
// suppressing full relationship and field-clause collection until the owning schema is
// parsed.
var embeddedCacheKey = "embedded_cache_store"

// ParseTagSetting splits a tag value by sep into uppercase, trimmed keys and their
// values. A key without a colon maps to itself; values retain embedded colons, escaped
// separators are rejoined, and escaped double quotes are unescaped. Repeated keys keep
// the last value. For example, ParseTagSetting("size:128;not null", ";") returns
// SIZE="128" and NOT NULL="NOT NULL".
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
		} else if k != "" {
			settings[k] = k
		}
	}

	return settings
}

// toColumns splits a nonempty comma-separated setting into trimmed field names,
// preserving order and any empty interior entries.
func toColumns(val string) (results []string) {
	if val != "" {
		for _, v := range strings.Split(val, ",") {
			results = append(results, strings.TrimSpace(v))
		}
	}
	return
}

// removeSettingFromTag removes matching settings from the gorm portion of a struct tag
// using case-insensitive regular expressions. names are trusted setting names used
// while synthesizing join-table fields, not arbitrary patterns from user input.
func removeSettingFromTag(tag reflect.StructTag, names ...string) reflect.StructTag {
	for _, name := range names {
		tag = reflect.StructTag(regexp.MustCompile(`(?i)(gorm:.*?)(`+name+`(:.*?)?)(;|("))`).ReplaceAllString(string(tag), "${1}${5}"))
	}
	return tag
}

// appendSettingFromTag prepends value to the gorm tag unless it already occurs as a
// substring. When adding a setting it rebuilds only the gorm tag, so callers must not
// rely on unrelated original tags surviving.
func appendSettingFromTag(tag reflect.StructTag, value string) reflect.StructTag {
	t := tag.Get("gorm")
	if strings.Contains(t, value) {
		return tag
	}
	return reflect.StructTag(fmt.Sprintf(`gorm:"%s;%s"`, value, t))
}

// GetRelationsValues traverses rels in order through a struct or slice reflect.Value and returns a
// slice of pointers to the final associated model type. Zero associations are skipped;
// non-pointer struct elements must be addressable. It reads already loaded Go values
// and does not query the database; an empty relation path returns an invalid
// reflect.Value.
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

// GetIdentityFieldValuesMap extracts ordered field tuples and groups model values by
// utils.ToStringKey. Structs and slice elements whose selected fields are all zero are
// omitted; repeated identities share one tuple and map to multiple values. Slice
// processing also skips repeated element identities.
//
// Inputs must have the expected model shape; map inputs must contain usable values for
// every requested database-column key; a missing key can panic before the Go-name
// fallback is attempted. Returned tuple order follows encounter order. These string
// keys are internal matching aids, not a collision-free external identifier encoding.
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

// GetIdentityFieldValuesMapFromValues extracts identities from each supplied value and
// concatenates their tuples and grouped models. Duplicate tuples across separate values
// are not globally deduplicated. Each input must satisfy GetIdentityFieldValuesMap's
// model-shape requirements.
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

// ToQueryValues converts foreign-key columns and identity tuples into the column/value
// operands for clause.IN. One key produces a single qualified column and flat values;
// composite keys produce ordered columns and tuple values. It assumes each tuple
// matches foreignKeys and neither validates identifiers nor executes SQL.
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

// embeddedNamer carries the owner table while delegating naming rules, ensuring
// flattened embedded fields use the containing model's table context.
type embeddedNamer struct {
	// Table is the containing model's resolved table name used by embedded schema parsing.
	Table string
	// Namer delegates column, index, constraint, and other naming rules to the owner.
	Namer
}
