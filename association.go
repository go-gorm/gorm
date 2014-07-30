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
		joinTable := association.Field.JoinTable
		// many to many
		if joinTable.joinTable != "" {
			whereSql := fmt.Sprintf("%v.%v IN (?)", joinTable.joinTable, scope.Quote(ToSnake(joinTable.associationForeignKey)))
			scope.db.Table(joinTable.joinTable).Where(whereSql, primaryKeys).Delete("")
		} else {
			association.err(errors.New("only many to many support delete"))
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

func (association *Association) Count() (count int) {
	joinTable := association.Field.JoinTable
	scope := association.Scope
	field := scope.IndirectValue().FieldByName(association.Column)
	fieldValue := field.Interface()

	// many to many
	if joinTable.joinTable != "" {
		newScope := scope.New(fieldValue)
		whereSql := fmt.Sprintf("%v.%v IN (SELECT %v.%v FROM %v WHERE %v.%v = ?)",
			newScope.QuotedTableName(),
			scope.Quote(newScope.PrimaryKey()),
			joinTable.joinTable,
			scope.Quote(joinTable.associationForeignKey),
			joinTable.joinTable,
			joinTable.joinTable,
			scope.Quote(joinTable.foreignKey))
		scope.db.Table(newScope.QuotedTableName()).Where(whereSql, scope.PrimaryKey()).Count(&count)
	}
	// association.Scope.related(value, association.Column)
	return -1
}
