package gorm

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/utils"
)

// Association Mode contains some helper methods to handle relationship things easily.
type Association struct {
	DB           *DB
	Relationship *schema.Relationship
	Error        error
}

func (db *DB) Association(column string) *Association {
	association := &Association{DB: db}
	table := db.Statement.Table

	if err := db.Statement.Parse(db.Statement.Model); err == nil {
		db.Statement.Table = table
		association.Relationship = db.Statement.Schema.Relationships.Relations[column]

		if association.Relationship == nil {
			association.Error = fmt.Errorf("%w: %v", ErrUnsupportedRelation, column)
		}

		db.Statement.ReflectValue = reflect.Indirect(reflect.ValueOf(db.Statement.Model))
	} else {
		association.Error = err
	}

	return association
}

func (association *Association) Find(out interface{}, conds ...interface{}) error {
	if association.Error == nil {
		var (
			queryConds = association.Relationship.ToQueryConditions(association.DB.Statement.ReflectValue)
			tx         = association.DB.Model(out).Table("")
		)

		if association.Relationship.JoinTable != nil {
			for _, queryClause := range association.Relationship.JoinTable.QueryClauses {
				tx.Clauses(queryClause)
			}

			tx.Clauses(clause.From{Joins: []clause.Join{{
				Table: clause.Table{Name: association.Relationship.JoinTable.Table},
				ON:    clause.Where{Exprs: queryConds},
			}}})
		} else {
			tx.Clauses(clause.Where{Exprs: queryConds})
		}

		association.Error = tx.Find(out, conds...).Error
	}

	return association.Error
}

func (association *Association) Append(values ...interface{}) error {
	if association.Error == nil {
		switch association.Relationship.Type {
		case schema.HasOne, schema.BelongsTo:
			if len(values) > 0 {
				association.Error = association.Replace(values...)
			}
		default:
			association.saveAssociation(false, values...)
		}
	}

	return association.Error
}

func (association *Association) Replace(values ...interface{}) error {
	if association.Error == nil {
		association.saveAssociation(true, values...)
		reflectValue := association.DB.Statement.ReflectValue
		rel := association.Relationship

		switch rel.Type {
		case schema.BelongsTo:
			if len(values) == 0 {
				updateMap := map[string]interface{}{}

				for _, ref := range rel.References {
					updateMap[ref.ForeignKey.DBName] = nil
				}

				association.DB.UpdateColumns(updateMap)
			}
		case schema.HasOne, schema.HasMany:
			var (
				tx             = association.DB
				primaryFields  []*schema.Field
				foreignKeys    []string
				updateMap      = map[string]interface{}{}
				relPrimaryKeys = []string{}
				relValues      = schema.GetRelationsValues(reflectValue, []*schema.Relationship{rel})
				modelValue     = reflect.New(rel.FieldSchema.ModelType).Interface()
			)

			for _, field := range rel.FieldSchema.PrimaryFields {
				relPrimaryKeys = append(relPrimaryKeys, field.DBName)
			}
			if _, qvs := schema.GetIdentityFieldValuesMap(relValues, rel.FieldSchema.PrimaryFields); len(qvs) > 0 {
				if column, values := schema.ToQueryValues(relPrimaryKeys, qvs); len(values) > 0 {
					tx = tx.Not(clause.IN{Column: column, Values: values})
				}
			}

			for _, ref := range rel.References {
				if ref.OwnPrimaryKey {
					primaryFields = append(primaryFields, ref.PrimaryKey)
					foreignKeys = append(foreignKeys, ref.ForeignKey.DBName)
					updateMap[ref.ForeignKey.DBName] = nil
				}
			}
			if _, qvs := schema.GetIdentityFieldValuesMap(reflectValue, primaryFields); len(qvs) > 0 {
				column, values := schema.ToQueryValues(foreignKeys, qvs)
				tx.Model(modelValue).Where(clause.IN{Column: column, Values: values}).UpdateColumns(updateMap)
			}
		case schema.Many2Many:
			var primaryFields, relPrimaryFields []*schema.Field
			var joinPrimaryKeys, joinRelPrimaryKeys []string
			var conds []clause.Expression

			for _, ref := range rel.References {
				if ref.PrimaryValue == "" {
					if ref.OwnPrimaryKey {
						primaryFields = append(primaryFields, ref.PrimaryKey)
						joinPrimaryKeys = append(joinPrimaryKeys, ref.ForeignKey.DBName)
					} else {
						relPrimaryFields = append(relPrimaryFields, ref.PrimaryKey)
						joinRelPrimaryKeys = append(joinRelPrimaryKeys, ref.ForeignKey.DBName)
					}
				} else {
					conds = append(conds, clause.Eq{Column: ref.ForeignKey.DBName, Value: ref.PrimaryValue})
				}
			}

			var (
				modelValue        = reflect.New(rel.JoinTable.ModelType).Interface()
				_, queryValues    = schema.GetIdentityFieldValuesMap(reflectValue, primaryFields)
				_, relQueryValues = schema.GetIdentityFieldValuesMapFromValues(values, relPrimaryFields)
			)

			if column, values := schema.ToQueryValues(joinPrimaryKeys, queryValues); len(values) > 0 {
				conds = append(conds, clause.IN{Column: column, Values: values})
			} else {
				return ErrorPrimaryKeyRequired
			}

			if relColumn, relValues := schema.ToQueryValues(joinRelPrimaryKeys, relQueryValues); len(relValues) > 0 {
				conds = append(conds, clause.Not(clause.IN{Column: relColumn, Values: relValues}))
			}

			association.DB.Where(clause.Where{Exprs: conds}).Model(nil).Delete(modelValue)
		}
	}
	return association.Error
}

func (association *Association) Delete(values ...interface{}) error {
	if association.Error == nil {
		var (
			reflectValue                 = association.DB.Statement.ReflectValue
			rel                          = association.Relationship
			tx                           = association.DB
			primaryFields, foreignFields []*schema.Field
			foreignKeys                  []string
			updateAttrs                  = map[string]interface{}{}
			conds                        []clause.Expression
		)

		for _, ref := range rel.References {
			if ref.PrimaryValue == "" {
				primaryFields = append(primaryFields, ref.PrimaryKey)
				foreignFields = append(foreignFields, ref.ForeignKey)
				foreignKeys = append(foreignKeys, ref.ForeignKey.DBName)
				updateAttrs[ref.ForeignKey.DBName] = nil
			} else {
				conds = append(conds, clause.Eq{Column: ref.ForeignKey.DBName, Value: ref.PrimaryValue})
			}
		}

		switch rel.Type {
		case schema.HasOne, schema.HasMany:
			var (
				modelValue        = reflect.New(rel.FieldSchema.ModelType).Interface()
				_, queryValues    = schema.GetIdentityFieldValuesMap(reflectValue, primaryFields)
				_, relQueryValues = schema.GetIdentityFieldValuesMapFromValues(values, rel.FieldSchema.PrimaryFields)
			)

			column, values := schema.ToQueryValues(foreignKeys, queryValues)
			conds = append(conds, clause.IN{Column: column, Values: values})
			relColumn, relValues := schema.ToQueryValues(rel.FieldSchema.PrimaryFieldDBNames, relQueryValues)
			conds = append(conds, clause.IN{Column: relColumn, Values: relValues})

			tx.Session(&Session{}).Model(modelValue).Clauses(conds...).UpdateColumns(updateAttrs)
		case schema.BelongsTo:
			var (
				modelValue        = reflect.New(rel.Schema.ModelType).Interface()
				_, queryValues    = schema.GetIdentityFieldValuesMap(reflectValue, rel.Schema.PrimaryFields)
				_, relQueryValues = schema.GetIdentityFieldValuesMapFromValues(values, primaryFields)
			)

			column, values := schema.ToQueryValues(rel.Schema.PrimaryFieldDBNames, queryValues)
			conds = append(conds, clause.IN{Column: column, Values: values})
			relColumn, relValues := schema.ToQueryValues(foreignKeys, relQueryValues)
			conds = append(conds, clause.IN{Column: relColumn, Values: relValues})

			tx.Session(&Session{}).Model(modelValue).Clauses(conds...).UpdateColumns(updateAttrs)
		case schema.Many2Many:
			var primaryFields, relPrimaryFields []*schema.Field
			var joinPrimaryKeys, joinRelPrimaryKeys []string

			for _, ref := range rel.References {
				if ref.PrimaryValue == "" {
					if ref.OwnPrimaryKey {
						primaryFields = append(primaryFields, ref.PrimaryKey)
						joinPrimaryKeys = append(joinPrimaryKeys, ref.ForeignKey.DBName)
					} else {
						relPrimaryFields = append(relPrimaryFields, ref.PrimaryKey)
						joinRelPrimaryKeys = append(joinRelPrimaryKeys, ref.ForeignKey.DBName)
					}
				} else {
					conds = append(conds, clause.Eq{Column: ref.ForeignKey.DBName, Value: ref.PrimaryValue})
				}
			}

			var (
				modelValue        = reflect.New(rel.JoinTable.ModelType).Interface()
				_, queryValues    = schema.GetIdentityFieldValuesMap(reflectValue, primaryFields)
				_, relQueryValues = schema.GetIdentityFieldValuesMapFromValues(values, relPrimaryFields)
			)

			if column, values := schema.ToQueryValues(joinPrimaryKeys, queryValues); len(values) > 0 {
				conds = append(conds, clause.IN{Column: column, Values: values})
			} else {
				return ErrorPrimaryKeyRequired
			}

			relColumn, relValues := schema.ToQueryValues(joinRelPrimaryKeys, relQueryValues)
			conds = append(conds, clause.IN{Column: relColumn, Values: relValues})

			tx.Where(clause.Where{Exprs: conds}).Model(nil).Delete(modelValue)
		}

		relValuesMap, _ := schema.GetIdentityFieldValuesMapFromValues(values, rel.FieldSchema.PrimaryFields)

		if tx.Error == nil {
			cleanUpDeletedRelations := func(data reflect.Value) {
				if _, zero := rel.Field.ValueOf(data); !zero {
					fieldValue := reflect.Indirect(rel.Field.ReflectValueOf(data))

					fieldValues := make([]interface{}, len(rel.FieldSchema.PrimaryFields))
					switch fieldValue.Kind() {
					case reflect.Slice, reflect.Array:
						validFieldValues := reflect.Zero(rel.Field.FieldType)
						for i := 0; i < fieldValue.Len(); i++ {
							for idx, field := range rel.FieldSchema.PrimaryFields {
								fieldValues[idx], _ = field.ValueOf(fieldValue.Index(i))
							}

							if _, ok := relValuesMap[utils.ToStringKey(fieldValues...)]; !ok {
								validFieldValues = reflect.Append(validFieldValues, fieldValue.Index(i))
							}
						}

						rel.Field.Set(data, validFieldValues.Interface())
					case reflect.Struct:
						for idx, field := range rel.FieldSchema.PrimaryFields {
							fieldValues[idx], _ = field.ValueOf(fieldValue)
						}
						if _, ok := relValuesMap[utils.ToStringKey(fieldValues...)]; ok {
							rel.Field.Set(data, reflect.Zero(rel.FieldSchema.ModelType).Interface())
							for _, ref := range rel.References {
								if ref.OwnPrimaryKey {
									ref.ForeignKey.Set(fieldValue, reflect.Zero(ref.ForeignKey.FieldType).Interface())
								} else if ref.PrimaryValue == "" {
									// FIXME
									ref.ForeignKey.Set(data, reflect.Zero(ref.ForeignKey.FieldType).Interface())
								}
							}
						}
					}
				}
			}

			switch reflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i < reflectValue.Len(); i++ {
					cleanUpDeletedRelations(reflect.Indirect(reflectValue.Index(i)))
				}
			case reflect.Struct:
				cleanUpDeletedRelations(reflectValue)
			}
		} else {
			association.Error = tx.Error
		}
	}
	return association.Error
}

func (association *Association) Clear() error {
	return association.Replace()
}

func (association *Association) Count() (count int64) {
	if association.Error == nil {
		var (
			conds      = association.Relationship.ToQueryConditions(association.DB.Statement.ReflectValue)
			modelValue = reflect.New(association.Relationship.FieldSchema.ModelType).Interface()
			tx         = association.DB.Model(modelValue)
		)

		if association.Relationship.JoinTable != nil {
			for _, queryClause := range association.Relationship.JoinTable.QueryClauses {
				tx.Clauses(queryClause)
			}

			tx.Clauses(clause.From{Joins: []clause.Join{{
				Table: clause.Table{Name: association.Relationship.JoinTable.Table},
				ON:    clause.Where{Exprs: conds},
			}}})
		} else {
			tx.Clauses(clause.Where{Exprs: conds})
		}

		association.Error = tx.Count(&count).Error
	}

	return
}

func (association *Association) saveAssociation(clear bool, values ...interface{}) {
	var (
		reflectValue = association.DB.Statement.ReflectValue
		assignBacks  = [][2]reflect.Value{}
		assignBack   = association.Relationship.Field.FieldType.Kind() == reflect.Struct
	)

	appendToRelations := func(source, rv reflect.Value, clear bool) {
		switch association.Relationship.Type {
		case schema.HasOne, schema.BelongsTo:
			switch rv.Kind() {
			case reflect.Slice, reflect.Array:
				if rv.Len() > 0 {
					association.Error = association.Relationship.Field.Set(source, rv.Index(0).Addr().Interface())
					if assignBack {
						assignBacks = append(assignBacks, [2]reflect.Value{source, rv.Index(0)})
					}
				}
			case reflect.Struct:
				association.Error = association.Relationship.Field.Set(source, rv.Addr().Interface())
				if assignBack {
					assignBacks = append(assignBacks, [2]reflect.Value{source, rv})
				}
			}
		case schema.HasMany, schema.Many2Many:
			elemType := association.Relationship.Field.IndirectFieldType.Elem()
			fieldValue := reflect.Indirect(association.Relationship.Field.ReflectValueOf(source))
			if clear {
				fieldValue = reflect.New(association.Relationship.Field.IndirectFieldType).Elem()
			}

			appendToFieldValues := func(ev reflect.Value) {
				if ev.Type().AssignableTo(elemType) {
					fieldValue = reflect.Append(fieldValue, ev)
				} else if ev.Type().Elem().AssignableTo(elemType) {
					fieldValue = reflect.Append(fieldValue, ev.Elem())
				} else {
					association.Error = fmt.Errorf("unsupported data type: %v for relation %v", ev.Type(), association.Relationship.Name)
				}
			}

			switch rv.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i < rv.Len(); i++ {
					appendToFieldValues(reflect.Indirect(rv.Index(i)).Addr())
				}
			case reflect.Struct:
				appendToFieldValues(rv.Addr())
			}

			if association.Error == nil {
				association.Error = association.Relationship.Field.Set(source, fieldValue.Interface())
			}
		}
	}

	selectedColumns := []string{association.Relationship.Name}
	for _, ref := range association.Relationship.References {
		if !ref.OwnPrimaryKey {
			selectedColumns = append(selectedColumns, ref.ForeignKey.Name)
		}
	}

	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		if len(values) != reflectValue.Len() {
			if clear && len(values) == 0 {
				for i := 0; i < reflectValue.Len(); i++ {
					association.Relationship.Field.Set(reflectValue.Index(i), reflect.New(association.Relationship.Field.IndirectFieldType).Interface())
					for _, ref := range association.Relationship.References {
						if !ref.OwnPrimaryKey && ref.PrimaryValue == "" {
							ref.ForeignKey.Set(reflectValue.Index(i), reflect.Zero(ref.ForeignKey.FieldType).Interface())
						}
					}
				}
				break
			}
			association.Error = errors.New("invalid association values, length doesn't match")
			return
		}

		for i := 0; i < reflectValue.Len(); i++ {
			appendToRelations(reflectValue.Index(i), reflect.Indirect(reflect.ValueOf(values[i])), clear)

			if len(values) > 0 {
				// TODO support save slice data, sql with case
				err := association.DB.Session(&Session{}).Select(selectedColumns).Model(nil).Save(reflectValue.Index(i).Addr().Interface()).Error
				association.DB.AddError(err)
			}
		}
	case reflect.Struct:
		if clear && len(values) == 0 {
			association.Relationship.Field.Set(reflectValue, reflect.New(association.Relationship.Field.IndirectFieldType).Interface())
			for _, ref := range association.Relationship.References {
				if !ref.OwnPrimaryKey && ref.PrimaryValue == "" {
					ref.ForeignKey.Set(reflectValue, reflect.Zero(ref.ForeignKey.FieldType).Interface())
				}
			}
		}

		for idx, value := range values {
			rv := reflect.Indirect(reflect.ValueOf(value))
			appendToRelations(reflectValue, rv, clear && idx == 0)
		}

		if len(values) > 0 {
			association.DB.Session(&Session{}).Select(selectedColumns).Model(nil).Save(reflectValue.Addr().Interface())
		}
	}

	for _, assignBack := range assignBacks {
		reflect.Indirect(assignBack[1]).Set(association.Relationship.Field.ReflectValueOf(assignBack[0]))
	}
}
