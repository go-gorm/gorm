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
	var (
		relationship = association.Field.Relationship
		scope        = association.Scope
		field        = association.Field.Field
		newDB        = scope.NewDB()
	)

	// Append new values
	association.Field.Set(reflect.Zero(association.Field.Field.Type()))
	association.Append(values...)

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
		var foreignKeyMap = map[string]interface{}{}
		for idx, foreignKey := range relationship.ForeignDBNames {
			foreignKeyMap[foreignKey] = nil
			if field, ok := scope.FieldByName(relationship.AssociationForeignFieldNames[idx]); ok {
				newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		// Relations except new created
		if len(values) > 0 {
			var newPrimaryKeys [][]interface{}
			var associationForeignFieldNames []string

			if relationship.Kind == "many2many" {
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
			sql := fmt.Sprintf("%v NOT IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(newPrimaryKeys))
			newDB = newDB.Where(sql, toQueryValues(newPrimaryKeys)...)
		}

		if relationship.Kind == "many_to_many" {
			association.setErr(relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, newDB, relationship))
		} else if relationship.Kind == "has_one" || relationship.Kind == "has_many" {
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

	if relationship.Kind == "many_to_many" {
		// many to many
		// current value's foreign keys
		for idx, foreignKey := range relationship.ForeignDBNames {
			if field, ok := scope.FieldByName(relationship.ForeignFieldNames[idx]); ok {
				newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		// deleting value's foreign keys
		primaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, values...)
		sql := fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(primaryKeys))
		newDB = newDB.Where(sql, toQueryValues(primaryKeys)...)

		if err := relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, newDB, relationship); err == nil {
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
			association.setErr(newDB.Model(scope.Value).UpdateColumn(foreignKeyMap).Error)
		} else if relationship.Kind == "has_one" || relationship.Kind == "has_many" {
			// find all relations
			primaryKeys := association.getPrimaryKeys(relationship.AssociationForeignFieldNames, scope.Value)
			newDB = newDB.Where(
				fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys)),
				toQueryValues(primaryKeys)...,
			)

			// only include those deleting relations
			var primaryFieldNames, primaryFieldDBNames []string
			for _, field := range scope.New(reflect.New(field.Type()).Interface()).Fields() {
				if field.IsPrimaryKey {
					primaryFieldNames = append(primaryFieldNames, field.Name)
					primaryFieldDBNames = append(primaryFieldDBNames, field.DBName)
				}
			}

			relationsPrimaryKeys := association.getPrimaryKeys(primaryFieldNames, values...)
			newDB = newDB.Where(
				fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, primaryFieldDBNames), toQueryMarks(relationsPrimaryKeys)),
				toQueryValues(relationsPrimaryKeys)...,
			)

			// set matched relation's foreign key to be null
			fieldValue := reflect.New(association.Field.Field.Type()).Interface()
			newDB.Model(fieldValue).UpdateColumn(foreignKeyMap)
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
