package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/utils"
)

// getRelationsValue get relations's value from a reflect value
func getRelationsValue(reflectValue reflect.Value, rels []*schema.Relationship) (reflectResults reflect.Value) {
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

func getIdentityFieldValuesMap(reflectValue reflect.Value, fields []*schema.Field) (map[string][]reflect.Value, [][]interface{}) {
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

func preloadData(tx *gorm.DB, resultSchema *schema.Schema, foreignKeys []string, foreignValues [][]interface{}, conds []interface{}) reflect.Value {
	slice := reflect.MakeSlice(reflect.SliceOf(resultSchema.ModelType), 0, 0)
	results := reflect.New(slice.Type())
	results.Elem().Set(slice)

	queryValues := make([]interface{}, len(foreignValues))
	if len(foreignKeys) == 1 {
		for idx, r := range foreignValues {
			queryValues[idx] = r[0]
		}
		tx.Where(clause.IN{Column: foreignKeys[0], Values: queryValues}).Find(results.Interface(), conds...)
	} else {
		for idx, r := range foreignValues {
			queryValues[idx] = r
		}
		tx.Where(clause.IN{Column: foreignKeys, Values: queryValues}).Find(results.Interface(), conds...)
	}

	return results.Elem()
}

func preload(db *gorm.DB, rels []*schema.Relationship, conds []interface{}) {
	var (
		reflectValue     = db.Statement.ReflectValue
		rel              = rels[len(rels)-1]
		tx               = db.Session(&gorm.Session{})
		relForeignKeys   []string
		relForeignFields []*schema.Field
		foreignFields    []*schema.Field
		foreignValues    [][]interface{}
		identityMap      = map[string][]reflect.Value{}
	)

	if len(rels) > 1 {
		reflectValue = getRelationsValue(reflectValue, rels[:len(rels)])
	}

	if rel.JoinTable != nil {
		var joinForeignFields, joinRelForeignFields []*schema.Field
		var joinForeignKeys []string
		for _, ref := range rel.References {
			if ref.OwnPrimaryKey {
				joinForeignKeys = append(joinForeignKeys, ref.ForeignKey.DBName)
				joinForeignFields = append(joinForeignFields, ref.ForeignKey)
				foreignFields = append(foreignFields, ref.PrimaryKey)
			} else if ref.PrimaryValue != "" {
				tx.Where(clause.Eq{Column: ref.ForeignKey.DBName, Value: ref.PrimaryValue})
			} else {
				joinRelForeignFields = append(joinRelForeignFields, ref.ForeignKey)
				relForeignKeys = append(relForeignKeys, ref.PrimaryKey.DBName)
				relForeignFields = append(relForeignFields, ref.PrimaryKey)
			}
		}

		joinIdentityMap, joinForeignValues := getIdentityFieldValuesMap(reflectValue, joinForeignFields)
		joinResults := preloadData(tx, rel.JoinTable, joinForeignKeys, joinForeignValues, nil)

		// convert join identity map to relation identity map
		fieldValues := make([]reflect.Value, len(foreignFields))
		joinFieldValues := make([]reflect.Value, len(joinForeignFields))
		for i := 0; i < joinResults.Len(); i++ {
			for idx, field := range foreignFields {
				fieldValues[idx] = field.ReflectValueOf(joinResults.Index(i))
			}

			for idx, field := range joinForeignFields {
				joinFieldValues[idx] = field.ReflectValueOf(joinResults.Index(i))
			}

			if results, ok := joinIdentityMap[utils.ToStringKey(fieldValues...)]; ok {
				identityMap[utils.ToStringKey(joinFieldValues...)] = results
			}
		}

		_, foreignValues = getIdentityFieldValuesMap(joinResults, joinRelForeignFields)
	} else {
		for _, ref := range rel.References {
			if ref.OwnPrimaryKey {
				relForeignKeys = append(relForeignKeys, ref.ForeignKey.DBName)
				relForeignFields = append(relForeignFields, ref.ForeignKey)
				foreignFields = append(foreignFields, ref.PrimaryKey)
			} else if ref.PrimaryValue != "" {
				tx.Where(clause.Eq{Column: ref.ForeignKey.DBName, Value: ref.PrimaryValue})
			} else {
				relForeignKeys = append(relForeignKeys, ref.PrimaryKey.DBName)
				relForeignFields = append(relForeignFields, ref.PrimaryKey)
				foreignFields = append(foreignFields, ref.ForeignKey)
			}
		}

		identityMap, foreignValues = getIdentityFieldValuesMap(reflectValue, foreignFields)
	}

	reflectResults := preloadData(tx, rel.FieldSchema, relForeignKeys, foreignValues, conds)

	fieldValues := make([]reflect.Value, len(foreignFields))
	for i := 0; i < reflectResults.Len(); i++ {
		for idx, field := range relForeignFields {
			fieldValues[idx] = field.ReflectValueOf(reflectResults.Index(i))
		}

		for _, data := range identityMap[utils.ToStringKey(fieldValues...)] {
			reflectFieldValue := reflect.Indirect(rel.Field.ReflectValueOf(data))
			switch reflectFieldValue.Kind() {
			case reflect.Struct:
				rel.Field.Set(data, reflectResults.Index(i).Interface())
			case reflect.Slice, reflect.Array:
				if reflectFieldValue.Type().Elem().Kind() == reflect.Ptr {
					rel.Field.Set(data, reflect.Append(reflectFieldValue, reflectResults.Index(i).Addr()).Interface())
				} else {
					rel.Field.Set(data, reflect.Append(reflectFieldValue, reflectResults.Index(i)).Interface())
				}
			}
		}
	}
}
