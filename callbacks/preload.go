package callbacks

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// parsePreloadMap extracts nested preloads. e.g.
//
//	// schema has a "k0" relation and a "k7.k8" embedded relation
//	parsePreloadMap(schema, map[string][]interface{}{
//		clause.Associations: {"arg1"},
//		"k1":                {"arg2"},
//		"k2.k3":             {"arg3"},
//		"k4.k5.k6":          {"arg4"},
//	})
//	// preloadMap is
//	map[string]map[string][]interface{}{
//		"k0": {},
//		"k7": {
//			"k8": {},
//		},
//		"k1": {},
//		"k2": {
//			"k3": {"arg3"},
//		},
//		"k4": {
//			"k5.k6": {"arg4"},
//		},
//	}
func parsePreloadMap(s *schema.Schema, preloads map[string][]interface{}) map[string]map[string][]interface{} {
	preloadMap := map[string]map[string][]interface{}{}
	setPreloadMap := func(name, value string, args []interface{}) {
		if _, ok := preloadMap[name]; !ok {
			preloadMap[name] = map[string][]interface{}{}
		}
		if value != "" {
			preloadMap[name][value] = args
		}
	}

	for name, args := range preloads {
		preloadFields := strings.Split(name, ".")
		value := strings.TrimPrefix(strings.TrimPrefix(name, preloadFields[0]), ".")
		if preloadFields[0] == clause.Associations {
			for _, relation := range s.Relationships.Relations {
				if relation.Schema == s {
					setPreloadMap(relation.Name, value, args)
				}
			}

			for embedded, embeddedRelations := range s.Relationships.EmbeddedRelations {
				for _, value := range embeddedValues(embeddedRelations) {
					setPreloadMap(embedded, value, args)
				}
			}
		} else {
			setPreloadMap(preloadFields[0], value, args)
		}
	}
	return preloadMap
}

func embeddedValues(embeddedRelations *schema.Relationships) []string {
	if embeddedRelations == nil {
		return nil
	}
	names := make([]string, 0, len(embeddedRelations.Relations)+len(embeddedRelations.EmbeddedRelations))
	for _, relation := range embeddedRelations.Relations {
		// skip first struct name
		names = append(names, strings.Join(relation.Field.BindNames[1:], "."))
	}
	for _, relations := range embeddedRelations.EmbeddedRelations {
		names = append(names, embeddedValues(relations)...)
	}
	return names
}

// preloadEntryPoint enters layer by layer. It will call real preload if it finds the right entry point.
// If the current relationship is embedded or joined, current query will be ignored.
//
//nolint:cyclop
func preloadEntryPoint(db *gorm.DB, joins []string, relationships *schema.Relationships, preloads map[string][]interface{}, associationsConds []interface{}) error {
	preloadMap := parsePreloadMap(db.Statement.Schema, preloads)

	// avoid random traversal of the map
	preloadNames := make([]string, 0, len(preloadMap))
	for key := range preloadMap {
		preloadNames = append(preloadNames, key)
	}
	sort.Strings(preloadNames)

	isJoined := func(name string) (joined bool, nestedJoins []string) {
		for _, join := range joins {
			if _, ok := relationships.Relations[join]; ok && name == join {
				joined = true
				continue
			}
			joinNames := strings.SplitN(join, ".", 2)
			if len(joinNames) == 2 {
				if _, ok := relationships.Relations[joinNames[0]]; ok && name == joinNames[0] {
					joined = true
					nestedJoins = append(nestedJoins, joinNames[1])
				}
			}
		}
		return joined, nestedJoins
	}

	for _, name := range preloadNames {
		if relations := relationships.EmbeddedRelations[name]; relations != nil {
			if err := preloadEntryPoint(db, joins, relations, preloadMap[name], associationsConds); err != nil {
				return err
			}
		} else if rel := relationships.Relations[name]; rel != nil {
			if joined, nestedJoins := isJoined(name); joined {
				reflectValue := rel.Field.ReflectValueOf(db.Statement.Context, db.Statement.ReflectValue)
				tx := preloadDB(db, reflectValue, reflectValue.Interface())
				if err := preloadEntryPoint(tx, nestedJoins, &tx.Statement.Schema.Relationships, preloadMap[name], associationsConds); err != nil {
					return err
				}
			} else {
				tx := db.Table("").Session(&gorm.Session{Context: db.Statement.Context, SkipHooks: db.Statement.SkipHooks})
				tx.Statement.ReflectValue = db.Statement.ReflectValue
				tx.Statement.Unscoped = db.Statement.Unscoped
				if err := preload(tx, rel, append(preloads[name], associationsConds...), preloadMap[name]); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("%s: %w for schema %s", name, gorm.ErrUnsupportedRelation, db.Statement.Schema.Name)
		}
	}
	return nil
}

func preloadDB(db *gorm.DB, reflectValue reflect.Value, dest interface{}) *gorm.DB {
	tx := db.Session(&gorm.Session{Context: db.Statement.Context, NewDB: true, SkipHooks: db.Statement.SkipHooks, Initialized: true})
	db.Statement.Settings.Range(func(k, v interface{}) bool {
		tx.Statement.Settings.Store(k, v)
		return true
	})

	if err := tx.Statement.Parse(dest); err != nil {
		tx.AddError(err)
		return tx
	}
	tx.Statement.ReflectValue = reflectValue
	tx.Statement.Unscoped = db.Statement.Unscoped
	return tx
}

func preload(tx *gorm.DB, rel *schema.Relationship, conds []interface{}, preloads map[string][]interface{}) error {
	var (
		reflectValue     = tx.Statement.ReflectValue
		relForeignKeys   []string
		relForeignFields []*schema.Field
		foreignFields    []*schema.Field
		foreignValues    [][]interface{}
		identityMap      = map[string][]reflect.Value{}
		inlineConds      []interface{}
	)

	if rel.JoinTable != nil {
		var (
			joinForeignFields    = make([]*schema.Field, 0, len(rel.References))
			joinRelForeignFields = make([]*schema.Field, 0, len(rel.References))
			joinForeignKeys      = make([]string, 0, len(rel.References))
		)

		for _, ref := range rel.References {
			if ref.OwnPrimaryKey {
				joinForeignKeys = append(joinForeignKeys, ref.ForeignKey.DBName)
				joinForeignFields = append(joinForeignFields, ref.ForeignKey)
				foreignFields = append(foreignFields, ref.PrimaryKey)
			} else if ref.PrimaryValue != "" {
				tx = tx.Where(clause.Eq{Column: ref.ForeignKey.DBName, Value: ref.PrimaryValue})
			} else {
				joinRelForeignFields = append(joinRelForeignFields, ref.ForeignKey)
				relForeignKeys = append(relForeignKeys, ref.PrimaryKey.DBName)
				relForeignFields = append(relForeignFields, ref.PrimaryKey)
			}
		}

		joinIdentityMap, joinForeignValues := schema.GetIdentityFieldValuesMap(tx.Statement.Context, reflectValue, foreignFields)
		if len(joinForeignValues) == 0 {
			return nil
		}

		joinResults := rel.JoinTable.MakeSlice().Elem()
		column, values := schema.ToQueryValues(clause.CurrentTable, joinForeignKeys, joinForeignValues)
		if err := tx.Where(clause.IN{Column: column, Values: values}).Find(joinResults.Addr().Interface()).Error; err != nil {
			return err
		}

		// convert join identity map to relation identity map
		fieldValues := make([]interface{}, len(joinForeignFields))
		joinFieldValues := make([]interface{}, len(joinRelForeignFields))
		for i := 0; i < joinResults.Len(); i++ {
			joinIndexValue := joinResults.Index(i)
			for idx, field := range joinForeignFields {
				fieldValues[idx], _ = field.ValueOf(tx.Statement.Context, joinIndexValue)
			}

			for idx, field := range joinRelForeignFields {
				joinFieldValues[idx], _ = field.ValueOf(tx.Statement.Context, joinIndexValue)
			}

			if results, ok := joinIdentityMap[utils.ToStringKey(fieldValues...)]; ok {
				joinKey := utils.ToStringKey(joinFieldValues...)
				identityMap[joinKey] = append(identityMap[joinKey], results...)
			}
		}

		_, foreignValues = schema.GetIdentityFieldValuesMap(tx.Statement.Context, joinResults, joinRelForeignFields)
	} else {
		for _, ref := range rel.References {
			if ref.OwnPrimaryKey {
				relForeignKeys = append(relForeignKeys, ref.ForeignKey.DBName)
				relForeignFields = append(relForeignFields, ref.ForeignKey)
				foreignFields = append(foreignFields, ref.PrimaryKey)
			} else if ref.PrimaryValue != "" {
				tx = tx.Where(clause.Eq{Column: ref.ForeignKey.DBName, Value: ref.PrimaryValue})
			} else {
				relForeignKeys = append(relForeignKeys, ref.PrimaryKey.DBName)
				relForeignFields = append(relForeignFields, ref.PrimaryKey)
				foreignFields = append(foreignFields, ref.ForeignKey)
			}
		}

		identityMap, foreignValues = schema.GetIdentityFieldValuesMap(tx.Statement.Context, reflectValue, foreignFields)
		if len(foreignValues) == 0 {
			return nil
		}
	}

	// nested preload
	for p, pvs := range preloads {
		tx = tx.Preload(p, pvs...)
	}

	reflectResults := rel.FieldSchema.MakeSlice().Elem()
	column, values := schema.ToQueryValues(clause.CurrentTable, relForeignKeys, foreignValues)

	if len(values) != 0 {
		for _, cond := range conds {
			if fc, ok := cond.(func(*gorm.DB) *gorm.DB); ok {
				tx = fc(tx)
			} else {
				inlineConds = append(inlineConds, cond)
			}
		}

		if err := tx.Where(clause.IN{Column: column, Values: values}).Find(reflectResults.Addr().Interface(), inlineConds...).Error; err != nil {
			return err
		}
	}

	fieldValues := make([]interface{}, len(relForeignFields))

	// clean up old values before preloading
	switch reflectValue.Kind() {
	case reflect.Struct:
		switch rel.Type {
		case schema.HasMany, schema.Many2Many:
			tx.AddError(rel.Field.Set(tx.Statement.Context, reflectValue, reflect.MakeSlice(rel.Field.IndirectFieldType, 0, 10).Interface()))
		default:
			tx.AddError(rel.Field.Set(tx.Statement.Context, reflectValue, reflect.New(rel.Field.FieldType).Interface()))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < reflectValue.Len(); i++ {
			switch rel.Type {
			case schema.HasMany, schema.Many2Many:
				tx.AddError(rel.Field.Set(tx.Statement.Context, reflectValue.Index(i), reflect.MakeSlice(rel.Field.IndirectFieldType, 0, 10).Interface()))
			default:
				tx.AddError(rel.Field.Set(tx.Statement.Context, reflectValue.Index(i), reflect.New(rel.Field.FieldType).Interface()))
			}
		}
	}

	for i := 0; i < reflectResults.Len(); i++ {
		elem := reflectResults.Index(i)
		for idx, field := range relForeignFields {
			fieldValues[idx], _ = field.ValueOf(tx.Statement.Context, elem)
		}

		datas, ok := identityMap[utils.ToStringKey(fieldValues...)]
		if !ok {
			return fmt.Errorf("failed to assign association %#v, make sure foreign fields exists", elem.Interface())
		}

		for _, data := range datas {
			reflectFieldValue := rel.Field.ReflectValueOf(tx.Statement.Context, data)
			if reflectFieldValue.Kind() == reflect.Ptr && reflectFieldValue.IsNil() {
				reflectFieldValue.Set(reflect.New(rel.Field.FieldType.Elem()))
			}

			reflectFieldValue = reflect.Indirect(reflectFieldValue)
			switch reflectFieldValue.Kind() {
			case reflect.Struct:
				tx.AddError(rel.Field.Set(tx.Statement.Context, data, elem.Interface()))
			case reflect.Slice, reflect.Array:
				if reflectFieldValue.Type().Elem().Kind() == reflect.Ptr {
					tx.AddError(rel.Field.Set(tx.Statement.Context, data, reflect.Append(reflectFieldValue, elem).Interface()))
				} else {
					tx.AddError(rel.Field.Set(tx.Statement.Context, data, reflect.Append(reflectFieldValue, elem.Elem()).Interface()))
				}
			}
		}
	}

	return tx.Error
}
