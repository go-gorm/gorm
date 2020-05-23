package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/utils"
)

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
		reflectValue = schema.GetRelationsValues(reflectValue, rels[:len(rels)-1])
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

		joinIdentityMap, joinForeignValues := schema.GetIdentityFieldValuesMap(reflectValue, foreignFields)
		if len(joinForeignValues) == 0 {
			return
		}

		joinResults := rel.JoinTable.MakeSlice().Elem()
		column, values := schema.ToQueryValues(joinForeignKeys, joinForeignValues)
		tx.Where(clause.IN{Column: column, Values: values}).Find(joinResults.Addr().Interface())

		// convert join identity map to relation identity map
		fieldValues := make([]interface{}, len(foreignFields))
		joinFieldValues := make([]interface{}, len(joinForeignFields))
		for i := 0; i < joinResults.Len(); i++ {
			for idx, field := range joinForeignFields {
				fieldValues[idx], _ = field.ValueOf(joinResults.Index(i))
			}

			for idx, field := range joinRelForeignFields {
				joinFieldValues[idx], _ = field.ValueOf(joinResults.Index(i))
			}

			if results, ok := joinIdentityMap[utils.ToStringKey(fieldValues...)]; ok {
				identityMap[utils.ToStringKey(joinFieldValues...)] = results
			}
		}

		_, foreignValues = schema.GetIdentityFieldValuesMap(joinResults, joinRelForeignFields)
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

		identityMap, foreignValues = schema.GetIdentityFieldValuesMap(reflectValue, foreignFields)
		if len(foreignValues) == 0 {
			return
		}
	}

	reflectResults := rel.FieldSchema.MakeSlice().Elem()
	column, values := schema.ToQueryValues(relForeignKeys, foreignValues)
	tx.Where(clause.IN{Column: column, Values: values}).Find(reflectResults.Addr().Interface(), conds...)

	fieldValues := make([]interface{}, len(foreignFields))
	for i := 0; i < reflectResults.Len(); i++ {
		for idx, field := range relForeignFields {
			fieldValues[idx], _ = field.ValueOf(reflectResults.Index(i))
		}

		for _, data := range identityMap[utils.ToStringKey(fieldValues...)] {
			reflectFieldValue := reflect.Indirect(rel.Field.ReflectValueOf(data))
			switch reflectFieldValue.Kind() {
			case reflect.Struct:
				rel.Field.Set(data, reflectResults.Index(i).Interface())
			case reflect.Slice, reflect.Array:
				if reflectFieldValue.Type().Elem().Kind() == reflect.Ptr {
					rel.Field.Set(data, reflect.Append(reflectFieldValue, reflectResults.Index(i)).Interface())
				} else {
					rel.Field.Set(data, reflect.Append(reflectFieldValue, reflectResults.Index(i).Elem()).Interface())
				}
			}
		}
	}
}
