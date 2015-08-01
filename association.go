package gorm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type Association struct {
	Scope  *Scope
	Column string
	Error  error
	Field  *Field
}

func (association *Association) setErr(err error) *Association {
	if err != nil {
		association.Error = err
	}
	return association
}

func (association *Association) Find(value interface{}) *Association {
	association.Scope.related(value, association.Column)
	return association.setErr(association.Scope.db.Error)
}

func (association *Association) Append(values ...interface{}) *Association {
	scope := association.Scope
	field := association.Field

	for _, value := range values {
		reflectvalue := reflect.Indirect(reflect.ValueOf(value))
		if reflectvalue.Kind() == reflect.Struct {
			field.Set(reflect.Append(field.Field, reflectvalue))
		} else if reflectvalue.Kind() == reflect.Slice {
			field.Set(reflect.AppendSlice(field.Field, reflectvalue))
		} else {
			association.setErr(errors.New("invalid association type"))
		}
	}
	scope.Search.Select(association.Column)
	scope.callCallbacks(scope.db.parent.callback.updates)
	return association.setErr(scope.db.Error)
}

func (association *Association) Delete(values ...interface{}) *Association {
	scope := association.Scope
	relationship := association.Field.Relationship

	// many to many
	if relationship.Kind == "many_to_many" {
		query := scope.NewDB()
		for idx, foreignKey := range relationship.ForeignDBNames {
			if field, ok := scope.FieldByName(relationship.ForeignFieldNames[idx]); ok {
				query = query.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		primaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, values...)
		sql := fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(primaryKeys))
		query = query.Where(sql, toQueryValues(primaryKeys)...)

		if err := relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, query, relationship); err == nil {
			leftValues := reflect.Zero(association.Field.Field.Type())
			for i := 0; i < association.Field.Field.Len(); i++ {
				reflectValue := association.Field.Field.Index(i)
				primaryKey := association.getPrimaryKeys(relationship.ForeignFieldNames, reflectValue.Interface())[0]
				var included = false
				for _, pk := range primaryKeys {
					if equalAsString(primaryKey, pk) {
						included = true
					}
				}
				if !included {
					leftValues = reflect.Append(leftValues, reflectValue)
				}
			}
			association.Field.Set(leftValues)
		}
	} else {
		association.setErr(errors.New("delete only support many to many"))
	}
	return association
}

func (association *Association) Replace(values ...interface{}) *Association {
	relationship := association.Field.Relationship
	scope := association.Scope
	if relationship.Kind == "many_to_many" {
		field := association.Field.Field

		oldPrimaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, field.Interface())
		association.Field.Set(reflect.Zero(association.Field.Field.Type()))
		association.Append(values...)
		newPrimaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, field.Interface())

		var addedPrimaryKeys = [][]interface{}{}
		for _, newKey := range newPrimaryKeys {
			hasEqual := false
			for _, oldKey := range oldPrimaryKeys {
				if equalAsString(newKey, oldKey) {
					hasEqual = true
					break
				}
			}
			if !hasEqual {
				addedPrimaryKeys = append(addedPrimaryKeys, newKey)
			}
		}

		for _, primaryKey := range association.getPrimaryKeys(relationship.AssociationForeignFieldNames, values...) {
			addedPrimaryKeys = append(addedPrimaryKeys, primaryKey)
		}

		if len(addedPrimaryKeys) > 0 {
			query := scope.NewDB()
			for idx, foreignKey := range relationship.ForeignDBNames {
				if field, ok := scope.FieldByName(relationship.ForeignFieldNames[idx]); ok {
					query = query.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
				}
			}

			sql := fmt.Sprintf("%v NOT IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(addedPrimaryKeys))
			query = query.Where(sql, toQueryValues(addedPrimaryKeys)...)
			association.setErr(relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, query, relationship))
		}
	} else {
		association.setErr(errors.New("replace only support many to many"))
	}
	return association
}

func (association *Association) Clear() *Association {
	relationship := association.Field.Relationship
	scope := association.Scope
	if relationship.Kind == "many_to_many" {
		query := scope.NewDB()
		for idx, foreignKey := range relationship.ForeignDBNames {
			if field, ok := scope.FieldByName(relationship.ForeignFieldNames[idx]); ok {
				query = query.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		if err := relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, query, relationship); err == nil {
			association.Field.Set(reflect.Zero(association.Field.Field.Type()))
		} else {
			association.setErr(err)
		}
	} else {
		association.setErr(errors.New("clear only support many to many"))
	}
	return association
}

func (association *Association) Count() int {
	count := -1
	relationship := association.Field.Relationship
	scope := association.Scope
	newScope := scope.New(association.Field.Field.Interface())

	if relationship.Kind == "many_to_many" {
		relationship.JoinTableHandler.JoinWith(relationship.JoinTableHandler, scope.NewDB(), association.Scope.Value).Table(newScope.TableName()).Count(&count)
	} else if relationship.Kind == "has_many" || relationship.Kind == "has_one" {
		query := scope.DB()
		for idx, foreignKey := range relationship.ForeignDBNames {
			if field, ok := scope.FieldByName(relationship.AssociationForeignDBNames[idx]); ok {
				query = query.Where(fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), scope.Quote(foreignKey)),
					field.Field.Interface())
			}
		}

		if relationship.PolymorphicType != "" {
			query = query.Where(fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), newScope.Quote(relationship.PolymorphicDBName)), scope.TableName())
		}
		query.Table(newScope.TableName()).Count(&count)
	} else if relationship.Kind == "belongs_to" {
		query := scope.DB()
		for idx, foreignKey := range relationship.ForeignDBNames {
			if field, ok := scope.FieldByName(relationship.AssociationForeignDBNames[idx]); ok {
				query = query.Where(fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), scope.Quote(foreignKey)),
					field.Field.Interface())
			}
		}
		query.Table(newScope.TableName()).Count(&count)
	}

	return count
}

func (association *Association) getPrimaryKeys(columns []string, values ...interface{}) [][]interface{} {
	results := [][]interface{}{}
	scope := association.Scope

	for _, value := range values {
		reflectValue := reflect.Indirect(reflect.ValueOf(value))
		if reflectValue.Kind() == reflect.Slice {
			for i := 0; i < reflectValue.Len(); i++ {
				primaryKeys := []interface{}{}
				newScope := scope.New(reflectValue.Index(i).Interface())
				for _, column := range columns {
					if field, ok := newScope.FieldByName(column); ok {
						primaryKeys = append(primaryKeys, field.Field.Interface())
					} else {
						primaryKeys = append(primaryKeys, "")
					}
				}
				results = append(results, primaryKeys)
			}
		} else if reflectValue.Kind() == reflect.Struct {
			newScope := scope.New(value)
			var primaryKeys []interface{}
			for _, column := range columns {
				if field, ok := newScope.FieldByName(column); ok {
					primaryKeys = append(primaryKeys, field.Field.Interface())
				} else {
					primaryKeys = append(primaryKeys, "")
				}
			}

			results = append(results, primaryKeys)
		}
	}
	return results
}

func toQueryMarks(primaryValues [][]interface{}) string {
	var results []string

	for _, primaryValue := range primaryValues {
		var marks []string
		for _,_ = range primaryValue {
			marks = append(marks, "?")
		}

		if len(marks) > 1 {
			results = append(results, fmt.Sprintf("(%v)", strings.Join(marks, ",")))
		} else {
			results = append(results, strings.Join(marks, ""))
		}
	}
	return strings.Join(results, ",")
}

func toQueryCondition(scope *Scope, columns []string) string {
	var newColumns []string
	for _, column := range columns {
		newColumns = append(newColumns, scope.Quote(column))
	}

	if len(columns) > 1 {
		return fmt.Sprintf("(%v)", strings.Join(newColumns, ","))
	} else {
		return strings.Join(columns, ",")
	}
}

func toQueryValues(primaryValues [][]interface{}) (values []interface{}) {
	for _, primaryValue := range primaryValues {
		for _, value := range primaryValue {
			values = append(values, value)
		}
	}
	return values
}
