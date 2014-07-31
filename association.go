package gorm

import (
	"errors"
	"fmt"
	"reflect"
)

type Association struct {
	Scope      *Scope
	PrimaryKey interface{}
	Column     string
	Error      error
	Field      *Field
}

func (association *Association) err(err error) *Association {
	if err != nil {
		association.Error = err
	}
	return association
}

func (association *Association) Find(value interface{}) *Association {
	association.Scope.related(value, association.Column)
	return association.err(association.Scope.db.Error)
}

func (association *Association) Append(values ...interface{}) *Association {
	scope := association.Scope
	field := scope.IndirectValue().FieldByName(association.Column)
	for _, value := range values {
		reflectvalue := reflect.ValueOf(value)
		if reflectvalue.Kind() == reflect.Ptr {
			if reflectvalue.Elem().Kind() == reflect.Struct {
				if field.Type().Elem().Kind() == reflect.Ptr {
					field.Set(reflect.Append(field, reflectvalue))
				} else if field.Type().Elem().Kind() == reflect.Struct {
					field.Set(reflect.Append(field, reflectvalue.Elem()))
				}
			} else if reflectvalue.Elem().Kind() == reflect.Slice {
				if field.Type().Elem().Kind() == reflect.Ptr {
					field.Set(reflect.AppendSlice(field, reflectvalue))
				} else if field.Type().Elem().Kind() == reflect.Struct {
					field.Set(reflect.AppendSlice(field, reflectvalue.Elem()))
				}
			}
		} else if reflectvalue.Kind() == reflect.Struct && field.Type().Elem().Kind() == reflect.Struct {
			field.Set(reflect.Append(field, reflectvalue))
		} else if reflectvalue.Kind() == reflect.Slice && field.Type().Elem() == reflectvalue.Type().Elem() {
			field.Set(reflect.AppendSlice(field, reflectvalue))
		} else {
			association.err(errors.New("invalid association type"))
		}
	}
	scope.callCallbacks(scope.db.parent.callback.updates)
	return association.err(scope.db.Error)
}

func (association *Association) getPrimaryKeys(values ...interface{}) []interface{} {
	primaryKeys := []interface{}{}
	scope := association.Scope

	for _, value := range values {
		reflectValue := reflect.ValueOf(value)
		if reflectValue.Kind() == reflect.Ptr {
			reflectValue = reflectValue.Elem()
		}
		if reflectValue.Kind() == reflect.Slice {
			for i := 0; i < reflectValue.Len(); i++ {
				newScope := scope.New(reflectValue.Index(i).Interface())
				primaryKey := newScope.PrimaryKeyValue()
				if !reflect.DeepEqual(reflect.ValueOf(primaryKey), reflect.Zero(reflect.ValueOf(primaryKey).Type())) {
					primaryKeys = append(primaryKeys, primaryKey)
				}
			}
		} else if reflectValue.Kind() == reflect.Struct {
			newScope := scope.New(value)
			primaryKey := newScope.PrimaryKeyValue()
			if !reflect.DeepEqual(reflect.ValueOf(primaryKey), reflect.Zero(reflect.ValueOf(primaryKey).Type())) {
				primaryKeys = append(primaryKeys, primaryKey)
			}
		}
	}
	return primaryKeys
}

func (association *Association) Delete(values ...interface{}) *Association {
	primaryKeys := association.getPrimaryKeys(values...)

	if len(primaryKeys) == 0 {
		association.err(errors.New("no primary key found"))
	} else {
		relationship := association.Field.Relationship
		// many to many
		if relationship.Kind == "many_to_many" {
			whereSql := fmt.Sprintf("%v.%v IN (?)", relationship.JoinTable, association.Scope.Quote(ToSnake(relationship.AssociationForeignKey)))
			association.Scope.db.Model("").Table(relationship.JoinTable).Where(whereSql, primaryKeys).Delete("")
		} else {
			association.err(errors.New("delete only support many to many"))
		}
	}
	return association
}

func (association *Association) Replace(values ...interface{}) *Association {
	relationship := association.Field.Relationship
	scope := association.Scope
	if relationship.Kind == "many_to_many" {
		field := scope.IndirectValue().FieldByName(association.Column)

		oldPrimaryKeys := association.getPrimaryKeys(field.Interface())
		association.Append(values...)
		newPrimaryKeys := association.getPrimaryKeys(field.Interface())

		var addedPrimaryKeys = []interface{}{}
		for _, new := range newPrimaryKeys {
			hasEqual := false
			for _, old := range oldPrimaryKeys {
				if reflect.DeepEqual(new, old) {
					hasEqual = true
					break
				}
			}
			if !hasEqual {
				addedPrimaryKeys = append(addedPrimaryKeys, new)
			}
		}
		for _, primaryKey := range association.getPrimaryKeys(values...) {
			addedPrimaryKeys = append(addedPrimaryKeys, primaryKey)
		}

		whereSql := fmt.Sprintf("%v.%v NOT IN (?)", relationship.JoinTable, scope.Quote(ToSnake(relationship.AssociationForeignKey)))
		scope.db.Model("").Table(relationship.JoinTable).Where(whereSql, addedPrimaryKeys).Delete("")
	} else {
		association.err(errors.New("replace only support many to many"))
	}
	return association
}

func (association *Association) Clear() *Association {
	relationship := association.Field.Relationship
	scope := association.Scope
	if relationship.Kind == "many_to_many" {
		whereSql := fmt.Sprintf("%v.%v = ?", relationship.JoinTable, scope.Quote(ToSnake(relationship.ForeignKey)))
		scope.db.Model("").Table(relationship.JoinTable).Where(whereSql, association.PrimaryKey).Delete("")
	} else {
		association.err(errors.New("clear only support many to many"))
	}
	return association
}

func (association *Association) Count() int {
	count := -1
	relationship := association.Field.Relationship
	scope := association.Scope
	field := scope.IndirectValue().FieldByName(association.Column)
	fieldValue := field.Interface()
	newScope := scope.New(fieldValue)

	if relationship.Kind == "many_to_many" {
		whereSql := fmt.Sprintf("%v.%v IN (SELECT %v.%v FROM %v WHERE %v.%v = ?)",
			newScope.QuotedTableName(),
			scope.Quote(newScope.PrimaryKey()),
			relationship.JoinTable,
			scope.Quote(ToSnake(relationship.AssociationForeignKey)),
			relationship.JoinTable,
			relationship.JoinTable,
			scope.Quote(ToSnake(relationship.ForeignKey)))
		scope.db.Model("").Table(newScope.QuotedTableName()).Where(whereSql, association.PrimaryKey).Count(&count)
	} else if relationship.Kind == "has_many" {
		whereSql := fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), newScope.Quote(ToSnake(relationship.ForeignKey)))
		scope.db.Model("").Table(newScope.QuotedTableName()).Where(whereSql, association.PrimaryKey).Count(&count)
	} else if relationship.Kind == "has_one" {
		whereSql := fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), relationship.ForeignKey)
		scope.db.Model("").Table(newScope.QuotedTableName()).Where(whereSql, association.PrimaryKey).Count(&count)
	} else if relationship.Kind == "belongs_to" {
		if v, ok := scope.FieldValueByName(association.Column); ok {
			whereSql := fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), relationship.ForeignKey)
			scope.db.Model("").Table(newScope.QuotedTableName()).Where(whereSql, v).Count(&count)
		}
	}

	return count
}
