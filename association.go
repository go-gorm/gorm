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

func (association *Association) saveAssociations(values ...interface{}) *Association {
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

		// value has to been saved for many2many
		if relationship.Kind == "many_to_many" {
			if scope.New(reflectValue.Interface()).PrimaryKeyZero() {
				association.setErr(scope.NewDB().Save(reflectValue.Interface()).Error)
			}
		}

		// Assign Fields
		var fieldType = field.Field.Type()
		var setFieldBackToValue, setSliceFieldBackToValue bool
		if reflectValue.Type().AssignableTo(fieldType) {
			field.Set(reflectValue)
		} else if reflectValue.Type().Elem().AssignableTo(fieldType) {
			// if field's type is struct, then need to set value back to argument after save
			setFieldBackToValue = true
			field.Set(reflectValue.Elem())
		} else if fieldType.Kind() == reflect.Slice {
			if reflectValue.Type().AssignableTo(fieldType.Elem()) {
				field.Set(reflect.Append(field.Field, reflectValue))
			} else if reflectValue.Type().Elem().AssignableTo(fieldType.Elem()) {
				// if field's type is slice of struct, then need to set value back to argument after save
				setSliceFieldBackToValue = true
				field.Set(reflect.Append(field.Field, reflectValue.Elem()))
			}
		}

		if relationship.Kind == "many_to_many" {
			association.setErr(relationship.JoinTableHandler.Add(relationship.JoinTableHandler, scope.NewDB(), scope.Value, reflectValue.Interface()))
		} else {
			association.setErr(scope.NewDB().Select(field.Name).Save(scope.Value).Error)

			if setFieldBackToValue {
				reflectValue.Elem().Set(field.Field)
			} else if setSliceFieldBackToValue {
				reflectValue.Elem().Set(field.Field.Index(field.Field.Len() - 1))
			}
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

func (association *Association) Append(values ...interface{}) *Association {
	if relationship := association.Field.Relationship; relationship.Kind == "has_one" {
		return association.Replace(values...)
	}
	return association.saveAssociations(values...)
}

func (association *Association) Replace(values ...interface{}) *Association {
	var (
		relationship = association.Field.Relationship
		scope        = association.Scope
		field        = association.Field.Field
		newDB        = scope.NewDB()
	)

	// Append new values
	association.Field.Set(reflect.Zero(association.Field.Field.Type()))
	association.saveAssociations(values...)

	// Belongs To
	if relationship.Kind == "belongs_to" {
		// Set foreign key to be null only when clearing value
		if len(values) == 0 {
			// Set foreign key to be nil
			var foreignKeyMap = map[string]interface{}{}
			for _, foreignKey := range relationship.ForeignDBNames {
				foreignKeyMap[foreignKey] = nil
			}
			association.setErr(newDB.Model(scope.Value).UpdateColumn(foreignKeyMap).Error)
		}
	} else {
		// Relations
		if relationship.PolymorphicDBName != "" {
			newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(relationship.PolymorphicDBName)), scope.TableName())
		}

		// Relations except new created
		if len(values) > 0 {
			var newPrimaryKeys [][]interface{}
			var associationForeignFieldNames []string

			if relationship.Kind == "many_to_many" {
				// If many to many relations, get it from foreign key
				associationForeignFieldNames = relationship.AssociationForeignFieldNames
			} else {
				// If other relations, get real primary keys
				for _, field := range scope.New(reflect.New(field.Type()).Interface()).Fields() {
					if field.IsPrimaryKey {
						associationForeignFieldNames = append(associationForeignFieldNames, field.Name)
					}
				}
			}

			newPrimaryKeys = association.getPrimaryKeys(associationForeignFieldNames, field.Interface())

			if len(newPrimaryKeys) > 0 {
				sql := fmt.Sprintf("%v NOT IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(newPrimaryKeys))
				newDB = newDB.Where(sql, toQueryValues(newPrimaryKeys)...)
			}
		}

		if relationship.Kind == "many_to_many" {
			for idx, foreignKey := range relationship.ForeignDBNames {
				if field, ok := scope.FieldByName(relationship.ForeignFieldNames[idx]); ok {
					newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
				}
			}

			association.setErr(relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, newDB, relationship))
		} else if relationship.Kind == "has_one" || relationship.Kind == "has_many" {
			var foreignKeyMap = map[string]interface{}{}
			for idx, foreignKey := range relationship.ForeignDBNames {
				foreignKeyMap[foreignKey] = nil
				if field, ok := scope.FieldByName(relationship.AssociationForeignFieldNames[idx]); ok {
					newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
				}
			}

			fieldValue := reflect.New(association.Field.Field.Type()).Interface()
			association.setErr(newDB.Model(fieldValue).UpdateColumn(foreignKeyMap).Error)
		}
	}
	return association
}

func (association *Association) Delete(values ...interface{}) *Association {
	var (
		relationship = association.Field.Relationship
		scope        = association.Scope
		field        = association.Field.Field
		newDB        = scope.NewDB()
	)

	if len(values) == 0 {
		return association
	}

	var deletingResourcePrimaryFieldNames, deletingResourcePrimaryDBNames []string
	for _, field := range scope.New(reflect.New(field.Type()).Interface()).Fields() {
		if field.IsPrimaryKey {
			deletingResourcePrimaryFieldNames = append(deletingResourcePrimaryFieldNames, field.Name)
			deletingResourcePrimaryDBNames = append(deletingResourcePrimaryDBNames, field.DBName)
		}
	}

	deletingPrimaryKeys := association.getPrimaryKeys(deletingResourcePrimaryFieldNames, values...)

	if relationship.Kind == "many_to_many" {
		// source value's foreign keys
		for idx, foreignKey := range relationship.ForeignDBNames {
			if field, ok := scope.FieldByName(relationship.ForeignFieldNames[idx]); ok {
				newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		// association value's foreign keys
		deletingPrimaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, values...)
		sql := fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(deletingPrimaryKeys))
		newDB = newDB.Where(sql, toQueryValues(deletingPrimaryKeys)...)

		association.setErr(relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, newDB, relationship))
	} else {
		var foreignKeyMap = map[string]interface{}{}
		for _, foreignKey := range relationship.ForeignDBNames {
			foreignKeyMap[foreignKey] = nil
		}

		if relationship.Kind == "belongs_to" {
			// find with deleting relation's foreign keys
			primaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, values...)
			newDB = newDB.Where(
				fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys)),
				toQueryValues(primaryKeys)...,
			)

			// set foreign key to be null
			modelValue := reflect.New(scope.GetModelStruct().ModelType).Interface()
			if results := newDB.Model(modelValue).UpdateColumn(foreignKeyMap); results.Error == nil {
				if results.RowsAffected > 0 {
					scope.updatedAttrsWithValues(foreignKeyMap, false)
				}
			} else {
				association.setErr(results.Error)
			}
		} else if relationship.Kind == "has_one" || relationship.Kind == "has_many" {
			// find all relations
			primaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, scope.Value)
			newDB = newDB.Where(
				fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys)),
				toQueryValues(primaryKeys)...,
			)

			// only include those deleting relations
			newDB = newDB.Where(
				fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, deletingResourcePrimaryDBNames), toQueryMarks(deletingPrimaryKeys)),
				toQueryValues(deletingPrimaryKeys)...,
			)

			// set matched relation's foreign key to be null
			fieldValue := reflect.New(association.Field.Field.Type()).Interface()
			association.setErr(newDB.Model(fieldValue).UpdateColumn(foreignKeyMap).Error)
		}
	}

	// Remove deleted records from field
	if association.Error == nil {
		if association.Field.Field.Kind() == reflect.Slice {
			leftValues := reflect.Zero(association.Field.Field.Type())

			for i := 0; i < association.Field.Field.Len(); i++ {
				reflectValue := association.Field.Field.Index(i)
				primaryKey := association.getPrimaryKeys(deletingResourcePrimaryFieldNames, reflectValue.Interface())[0]
				var included = false
				for _, pk := range deletingPrimaryKeys {
					if equalAsString(primaryKey, pk) {
						included = true
					}
				}
				if !included {
					leftValues = reflect.Append(leftValues, reflectValue)
				}
			}

			association.Field.Set(leftValues)
		} else if association.Field.Field.Kind() == reflect.Struct {
			primaryKey := association.getPrimaryKeys(deletingResourcePrimaryFieldNames, association.Field.Field.Interface())[0]
			for _, pk := range deletingPrimaryKeys {
				if equalAsString(primaryKey, pk) {
					association.Field.Set(reflect.Zero(association.Field.Field.Type()))
					break
				}
			}
		}
	}

	return association
}

func (association *Association) Clear() *Association {
	return association.Replace()
}

func (association *Association) Count() int {
	var (
		count        = 0
		relationship = association.Field.Relationship
		scope        = association.Scope
		fieldValue   = association.Field.Field.Interface()
		newScope     = scope.New(fieldValue)
	)

	if relationship.Kind == "many_to_many" {
		relationship.JoinTableHandler.JoinWith(relationship.JoinTableHandler, scope.DB(), association.Scope.Value).Model(fieldValue).Count(&count)
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
		query.Model(fieldValue).Count(&count)
	} else if relationship.Kind == "belongs_to" {
		query := scope.DB()
		for idx, primaryKey := range relationship.AssociationForeignDBNames {
			if field, ok := scope.FieldByName(relationship.ForeignDBNames[idx]); ok {
				query = query.Where(fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), scope.Quote(primaryKey)),
					field.Field.Interface())
			}
		}
		query.Model(fieldValue).Count(&count)
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
		return strings.Join(newColumns, ",")
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
