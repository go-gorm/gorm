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
	relationship := association.Field.Relationship

	saveAssociation := func(reflectValue reflect.Value) {
		// value has to been pointer
		if reflectValue.Kind() != reflect.Ptr {
			reflectPtr := reflect.New(reflectValue.Type())
			reflectPtr.Elem().Set(reflectValue)
			reflectValue = reflectPtr
		}

		// value has to been saved
		if scope.New(reflectValue.Interface()).PrimaryKeyZero() {
			scope.NewDB().Save(reflectValue.Interface())
		}

		// Assign Fields
		fieldType := field.Field.Type()
		if reflectValue.Type().AssignableTo(fieldType) {
			field.Set(reflectValue)
		} else if reflectValue.Type().Elem().AssignableTo(fieldType) {
			field.Set(reflectValue.Elem())
		} else if fieldType.Kind() == reflect.Slice {
			if reflectValue.Type().AssignableTo(fieldType.Elem()) {
				field.Set(reflect.Append(field.Field, reflectValue))
			} else if reflectValue.Type().Elem().AssignableTo(fieldType.Elem()) {
				field.Set(reflect.Append(field.Field, reflectValue.Elem()))
			}
		}

		if relationship.Kind == "many_to_many" {
			association.setErr(relationship.JoinTableHandler.Add(relationship.JoinTableHandler, scope.NewDB(), scope.Value, reflectValue.Interface()))
		} else {
			association.setErr(scope.NewDB().Select(field.Name).Save(scope.Value).Error)
		}
	}

	for _, value := range values {
		reflectValue := reflect.ValueOf(value)
		indirectReflectValue := reflect.Indirect(reflectValue)
		if indirectReflectValue.Kind() == reflect.Struct {
			saveAssociation(reflectValue)
		} else if indirectReflectValue.Kind() == reflect.Slice {
			for i := 0; i < indirectReflectValue.Len(); i++ {
				saveAssociation(indirectReflectValue.Index(i))
			}
		} else {
			association.setErr(errors.New("invalid value type"))
		}
	}
	return association
}

func (association *Association) Replace(values ...interface{}) *Association {
	relationship := association.Field.Relationship
	scope := association.Scope
	field := association.Field.Field

	// get old primary keys
	oldPrimaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, field.Interface())

	// append new values
	association.Field.Set(reflect.Zero(association.Field.Field.Type()))
	association.Append(values...)

	// get new primary keys
	var newPrimaryKeys [][]interface{}
	if len(values) > 0 {
		newPrimaryKeys = association.getPrimaryKeys(relationship.AssociationForeignFieldNames, field.Interface())
	}

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

	query := scope.NewDB()
	var foreignKeyMap = map[string]interface{}{}

	if relationship.Kind == "belongs_to" {
		for idx, foreignKey := range relationship.AssociationForeignDBNames {
			if field, ok := scope.FieldByName(relationship.AssociationForeignFieldNames[idx]); ok {
				query = query.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		for _, foreignKey := range relationship.ForeignDBNames {
			foreignKeyMap[foreignKey] = nil
		}

		if len(addedPrimaryKeys) > 0 {
			sql := fmt.Sprintf("%v NOT IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(addedPrimaryKeys))
			query = query.Where(sql, toQueryValues(addedPrimaryKeys)...)
		}

		modelValue := scope.Value
		// if replacing with a new value, don't reset current foreign key
		// if clearing foreign value, then reset the foreign key to null
		if len(values) > 0 {
			modelValue = reflect.New(scope.GetModelStruct().ModelType).Interface()
		}
		association.setErr(query.Model(modelValue).UpdateColumn(foreignKeyMap).Error)
	} else {
		for idx, foreignKey := range relationship.ForeignDBNames {
			foreignKeyMap[foreignKey] = nil
			if field, ok := scope.FieldByName(relationship.AssociationForeignFieldNames[idx]); ok {
				query = query.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		if len(addedPrimaryKeys) > 0 {
			sql := fmt.Sprintf("%v NOT IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(addedPrimaryKeys))
			query = query.Where(sql, toQueryValues(addedPrimaryKeys)...)
		}

		if relationship.Kind == "many_to_many" {
			association.setErr(relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, query, relationship))
		} else if relationship.Kind == "has_one" || relationship.Kind == "has_many" {
			fieldValue := reflect.New(association.Field.Field.Type()).Interface()
			association.setErr(query.Model(fieldValue).UpdateColumn(foreignKeyMap).Error)
		}
	}
	return association
}

func (association *Association) Delete(values ...interface{}) *Association {
	scope := association.Scope
	query := scope.NewDB()
	relationship := association.Field.Relationship

	if len(values) == 0 {
		return association
	}

	// many to many
	if relationship.Kind == "many_to_many" {
		// current value's foreign keys
		for idx, foreignKey := range relationship.ForeignDBNames {
			if field, ok := scope.FieldByName(relationship.ForeignFieldNames[idx]); ok {
				query = query.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		// deleting value's foreign keys
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
		association.Field.Set(reflect.Zero(association.Field.Field.Type()))

		if relationship.Kind == "belongs_to" {
			var foreignKeyMap = map[string]interface{}{}
			for _, foreignKey := range relationship.ForeignDBNames {
				foreignKeyMap[foreignKey] = nil
			}
			primaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, values...)
			sql := fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys))

			association.setErr(query.Model(scope.Value).Where(sql, toQueryValues(primaryKeys)...).UpdateColumn(foreignKeyMap).Error)
		} else if relationship.Kind == "has_one" || relationship.Kind == "has_many" {
			var foreignKeyMap = map[string]interface{}{}
			for _, foreignKey := range relationship.ForeignDBNames {
				foreignKeyMap[foreignKey] = nil
			}

			primaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, scope.Value)
			sql := fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys))

			var primaryFieldNames, primaryFieldDBNames []string
			for _, field := range scope.New(values[0]).Fields() {
				if field.IsPrimaryKey {
					primaryFieldNames = append(primaryFieldNames, field.Name)
					primaryFieldDBNames = append(primaryFieldDBNames, field.DBName)
				}
			}
			relationsPrimaryKeys := association.getPrimaryKeys(primaryFieldNames, values...)
			sql += fmt.Sprintf(" AND %v IN (%v)", toQueryCondition(scope, primaryFieldDBNames), toQueryMarks(relationsPrimaryKeys))

			query.Model(association.Field.Field.Interface()).Where(sql, append(toQueryValues(primaryKeys), toQueryValues(relationsPrimaryKeys)...)...).UpdateColumn(foreignKeyMap)
		}
	}
	return association
}

func (association *Association) Clear() *Association {
	return association.Replace()
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

func (association *Association) getPrimaryKeys(columns []string, values ...interface{}) (results [][]interface{}) {
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

	return
}

func toQueryMarks(primaryValues [][]interface{}) string {
	var results []string

	for _, primaryValue := range primaryValues {
		var marks []string
		for _, _ = range primaryValue {
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
