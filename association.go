package gorm

import (
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

	if err := db.Statement.Parse(db.Statement.Model); err == nil {
		association.Relationship = db.Statement.Schema.Relationships.Relations[column]

		if association.Relationship == nil {
			association.Error = fmt.Errorf("%w: %v", ErrUnsupportedRelation, column)
		}
	} else {
		association.Error = err
	}

	return association
}

func (association *Association) Find(out interface{}, conds ...interface{}) error {
	if association.Error == nil {
	}

	return association.Error
}

func (association *Association) Append(values ...interface{}) error {
	return association.Error
}

func (association *Association) Replace(values ...interface{}) error {
	return association.Error
}

func (association *Association) Delete(values ...interface{}) error {
	if association.Error == nil {
		var (
			tx           = association.DB
			rel          = association.Relationship
			reflectValue = tx.Statement.ReflectValue
			conds        = rel.ToQueryConditions(reflectValue)
			relFields    []*schema.Field
			foreignKeys  []string
			updateAttrs  = map[string]interface{}{}
		)

		for _, ref := range rel.References {
			if ref.PrimaryValue == "" {
				if rel.JoinTable == nil || !ref.OwnPrimaryKey {
					if ref.OwnPrimaryKey {
						relFields = append(relFields, ref.ForeignKey)
					} else {
						relFields = append(relFields, ref.PrimaryKey)
					}

					foreignKeys = append(foreignKeys, ref.ForeignKey.DBName)
					updateAttrs[ref.ForeignKey.DBName] = nil
				}
			}
		}

		relValuesMap, relQueryValues := schema.GetIdentityFieldValuesMapFromValues(values, relFields)
		column, values := schema.ToQueryValues(foreignKeys, relQueryValues)
		tx.Where(clause.IN{Column: column, Values: values})

		switch association.Relationship.Type {
		case schema.HasOne, schema.HasMany:
			modelValue := reflect.New(rel.FieldSchema.ModelType).Interface()
			tx.Model(modelValue).Clauses(clause.Where{Exprs: conds}).UpdateColumns(updateAttrs)
		case schema.BelongsTo:
			tx.Clauses(clause.Where{Exprs: conds}).UpdateColumns(updateAttrs)
		case schema.Many2Many:
			modelValue := reflect.New(rel.JoinTable.ModelType).Interface()
			tx.Clauses(clause.Where{Exprs: conds}).Delete(modelValue)
		}

		if tx.Error == nil {
			cleanUpDeletedRelations := func(data reflect.Value) {
				if _, zero := rel.Field.ValueOf(data); !zero {
					fieldValue := reflect.Indirect(rel.Field.ReflectValueOf(data))

					fieldValues := make([]reflect.Value, len(relFields))
					switch fieldValue.Kind() {
					case reflect.Slice, reflect.Array:
						validFieldValues := reflect.Zero(rel.Field.FieldType)
						for i := 0; i < fieldValue.Len(); i++ {
							for idx, field := range relFields {
								fieldValues[idx] = field.ReflectValueOf(fieldValue.Index(i))
							}

							if _, ok := relValuesMap[utils.ToStringKey(fieldValues...)]; !ok {
								validFieldValues = reflect.Append(validFieldValues, fieldValue.Index(i))
							}
						}

						rel.Field.Set(data, validFieldValues)
					case reflect.Struct:
						for idx, field := range relFields {
							fieldValues[idx] = field.ReflectValueOf(data)
						}
						if _, ok := relValuesMap[utils.ToStringKey(fieldValues...)]; ok {
							rel.Field.Set(data, reflect.Zero(rel.FieldSchema.ModelType))
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

func (association *Association) Count() (count int) {
	if association.Error == nil {
		var (
			tx    = association.DB
			conds = association.Relationship.ToQueryConditions(tx.Statement.ReflectValue)
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
