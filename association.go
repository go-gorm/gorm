package gorm

import (
	"fmt"
	"reflect"
)

// Association Association Mode contains some helper methods to handle relationship things easily.
type Association struct {
	Scope  *Scope
	Column string
	Error  error
	Field  *Field
}

// Find find out all related associations
func (association *Association) Find(value interface{}) *Association {
	association.Scope.related(value, association.Column)
	return association.setErr(association.Scope.db.Error)
}

// Append append new associations for many2many, has_many, will replace current association for has_one, belongs_to
func (association *Association) Append(values ...interface{}) *Association {
	if relationship := association.Field.Relationship; relationship.Kind == "has_one" {
		return association.Replace(values...)
	}
	return association.saveAssociations(values...)
}

// Replace replace current associations with new one
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
		// Set foreign key to be null when clearing value (length equals 0)
		if len(values) == 0 {
			// Set foreign key to be nil
			var foreignKeyMap = map[string]interface{}{}
			for _, foreignKey := range relationship.ForeignDBNames {
				foreignKeyMap[foreignKey] = nil
			}
			association.setErr(newDB.Model(scope.Value).UpdateColumn(foreignKeyMap).Error)
		}
	} else {
		// Polymorphic Relations
		if relationship.PolymorphicDBName != "" {
			newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(relationship.PolymorphicDBName)), scope.TableName())
		}

		// Delete Relations except new created
		if len(values) > 0 {
			var associationForeignFieldNames []string
			if relationship.Kind == "many_to_many" {
				// if many to many relations, get association fields name from association foreign keys
				associationFields := scope.New(reflect.New(field.Type()).Interface()).Fields()
				for _, dbName := range relationship.AssociationForeignFieldNames {
					associationForeignFieldNames = append(associationForeignFieldNames, associationFields[dbName].Name)
				}
			} else {
				// If other relations, use primary keys
				for _, field := range scope.New(reflect.New(field.Type()).Interface()).PrimaryFields() {
					associationForeignFieldNames = append(associationForeignFieldNames, field.Name)
				}
			}

			newPrimaryKeys := scope.getColumnAsArray(associationForeignFieldNames, field.Interface())

			if len(newPrimaryKeys) > 0 {
				sql := fmt.Sprintf("%v NOT IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(newPrimaryKeys))
				newDB = newDB.Where(sql, toQueryValues(newPrimaryKeys)...)
			}
		}

		if relationship.Kind == "many_to_many" {
			// if many to many relations, delete related relations from join table

			// get source fields name from source foreign keys
			var (
				sourceFields            = scope.Fields()
				sourceForeignFieldNames []string
			)

			for _, dbName := range relationship.ForeignFieldNames {
				sourceForeignFieldNames = append(sourceForeignFieldNames, sourceFields[dbName].Name)
			}

			if sourcePrimaryKeys := scope.getColumnAsArray(sourceForeignFieldNames, scope.Value); len(sourcePrimaryKeys) > 0 {
				newDB = newDB.Where(fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(sourcePrimaryKeys)), toQueryValues(sourcePrimaryKeys)...)

				association.setErr(relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, newDB, relationship))
			}
		} else if relationship.Kind == "has_one" || relationship.Kind == "has_many" {
			// has_one or has_many relations, set foreign key to be nil (TODO or delete them?)
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

// Delete remove relationship between source & passed arguments, but won't delete those arguments
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
	for _, field := range scope.New(reflect.New(field.Type()).Interface()).PrimaryFields() {
		deletingResourcePrimaryFieldNames = append(deletingResourcePrimaryFieldNames, field.Name)
		deletingResourcePrimaryDBNames = append(deletingResourcePrimaryDBNames, field.DBName)
	}

	deletingPrimaryKeys := scope.getColumnAsArray(deletingResourcePrimaryFieldNames, values...)

	if relationship.Kind == "many_to_many" {
		// source value's foreign keys
		for idx, foreignKey := range relationship.ForeignDBNames {
			if field, ok := scope.FieldByName(relationship.ForeignFieldNames[idx]); ok {
				newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		// get association's foreign fields name
		var associationFields = scope.New(reflect.New(field.Type()).Interface()).Fields()
		var associationForeignFieldNames []string
		for _, associationDBName := range relationship.AssociationForeignFieldNames {
			associationForeignFieldNames = append(associationForeignFieldNames, associationFields[associationDBName].Name)
		}

		// association value's foreign keys
		deletingPrimaryKeys := scope.getColumnAsArray(associationForeignFieldNames, values...)
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
			primaryKeys := scope.getColumnAsArray(relationship.AssociationForeignFieldNames, values...)
			newDB = newDB.Where(
				fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys)),
				toQueryValues(primaryKeys)...,
			)

			// set foreign key to be null if there are some records affected
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
			primaryKeys := scope.getColumnAsArray(relationship.AssociationForeignFieldNames, scope.Value)
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

	// Remove deleted records from source's field
	if association.Error == nil {
		if association.Field.Field.Kind() == reflect.Slice {
			leftValues := reflect.Zero(association.Field.Field.Type())

			for i := 0; i < association.Field.Field.Len(); i++ {
				reflectValue := association.Field.Field.Index(i)
				primaryKey := scope.getColumnAsArray(deletingResourcePrimaryFieldNames, reflectValue.Interface())[0]
				var isDeleted = false
				for _, pk := range deletingPrimaryKeys {
					if equalAsString(primaryKey, pk) {
						isDeleted = true
						break
					}
				}
				if !isDeleted {
					leftValues = reflect.Append(leftValues, reflectValue)
				}
			}

			association.Field.Set(leftValues)
		} else if association.Field.Field.Kind() == reflect.Struct {
			primaryKey := scope.getColumnAsArray(deletingResourcePrimaryFieldNames, association.Field.Field.Interface())[0]
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

// Clear remove relationship between source & current associations, won't delete those associations
func (association *Association) Clear() *Association {
	return association.Replace()
}

// Count return the count of current associations
func (association *Association) Count() int {
	var (
		count        = 0
		relationship = association.Field.Relationship
		scope        = association.Scope
		fieldValue   = association.Field.Field.Interface()
		query        = scope.DB()
	)

	if relationship.Kind == "many_to_many" {
		query = relationship.JoinTableHandler.JoinWith(relationship.JoinTableHandler, scope.DB(), association.Scope.Value)
	} else if relationship.Kind == "has_many" || relationship.Kind == "has_one" {
		primaryKeys := scope.getColumnAsArray(relationship.AssociationForeignFieldNames, scope.Value)
		query = query.Where(
			fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys)),
			toQueryValues(primaryKeys)...,
		)
	} else if relationship.Kind == "belongs_to" {
		primaryKeys := scope.getColumnAsArray(relationship.ForeignFieldNames, scope.Value)
		query = query.Where(
			fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(primaryKeys)),
			toQueryValues(primaryKeys)...,
		)
	}

	if relationship.PolymorphicType != "" {
		query = query.Where(
			fmt.Sprintf("%v.%v = ?", scope.New(fieldValue).QuotedTableName(), scope.Quote(relationship.PolymorphicDBName)),
			scope.TableName(),
		)
	}

	query.Model(fieldValue).Count(&count)
	return count
}
