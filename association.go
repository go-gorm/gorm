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

func (association *Association) Delete(values ...interface{}) *Association {
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

	if len(primaryKeys) == 0 {
		association.err(errors.New("no primary key found"))
	} else {
		relationship := association.Field.Relationship
		// many to many
		if relationship.kind == "many_to_many" {
			whereSql := fmt.Sprintf("%v.%v IN (?)", relationship.joinTable, scope.Quote(ToSnake(relationship.associationForeignKey)))
			scope.db.Table(relationship.joinTable).Where(whereSql, primaryKeys).Delete("")
		} else {
			association.err(errors.New("delete only support many to many"))
		}
	}
	return association
}

func (association *Association) Replace(values interface{}) *Association {
	return association
}

func (association *Association) Clear(value interface{}) *Association {
	return association
}

func (association *Association) Count() int {
	count := -1
	relationship := association.Field.Relationship
	scope := association.Scope
	field := scope.IndirectValue().FieldByName(association.Column)
	fieldValue := field.Interface()
	newScope := scope.New(fieldValue)

	if relationship.kind == "many_to_many" {
		whereSql := fmt.Sprintf("%v.%v IN (SELECT %v.%v FROM %v WHERE %v.%v = ?)",
			newScope.QuotedTableName(),
			scope.Quote(newScope.PrimaryKey()),
			relationship.joinTable,
			scope.Quote(ToSnake(relationship.associationForeignKey)),
			relationship.joinTable,
			relationship.joinTable,
			scope.Quote(ToSnake(relationship.foreignKey)))
		scope.db.Table(newScope.QuotedTableName()).Where(whereSql, association.PrimaryKey).NewScope("").count(&count)
	} else if relationship.kind == "has_many" {
		whereSql := fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), newScope.Quote(ToSnake(relationship.foreignKey)))
		scope.db.Table(newScope.QuotedTableName()).Where(whereSql, association.PrimaryKey).NewScope("").count(&count)
	} else if relationship.kind == "has_one" {
		whereSql := fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), relationship.foreignKey)
		scope.db.Table(newScope.QuotedTableName()).Where(whereSql, association.PrimaryKey).NewScope("").count(&count)
	} else if relationship.kind == "belongs_to" {
		if v, ok := scope.FieldByName(association.Column); ok {
			whereSql := fmt.Sprintf("%v.%v = ?", newScope.QuotedTableName(), relationship.foreignKey)
			scope.db.Table(newScope.QuotedTableName()).Where(whereSql, v).NewScope("").count(&count)
		}
	}

	return count
}
