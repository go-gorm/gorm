package gorm

import (
	"errors"
	"fmt"
	"reflect"
)

type Association struct {
	Scope  *Scope
	Column string
	Error  error
}

func (association *Association) Find(value interface{}) *Association {
	scope := association.Scope
	primaryKey := scope.PrimaryKeyValue()
	if reflect.DeepEqual(reflect.ValueOf(primaryKey), reflect.Zero(reflect.ValueOf(primaryKey).Type())) {
		association.Error = errors.New("primary key can't be nil")
	}

	scopeType := scope.IndirectValue().Type()
	if f, ok := scopeType.FieldByName(SnakeToUpperCamel(association.Column)); ok {
		field := scope.fieldFromStruct(f)
		joinTable := field.JoinTable
		if joinTable != nil && joinTable.foreignKey != "" {
			if joinTable.joinTable != "" {
				newScope := scope.New(value)
				joinSql := fmt.Sprintf(
					"INNER JOIN %v ON %v.%v = %v.%v",
					scope.Quote(joinTable.joinTable),
					scope.Quote(joinTable.joinTable),
					scope.Quote(ToSnake(joinTable.associationForeignKey)),
					newScope.QuotedTableName(),
					scope.Quote(newScope.PrimaryKey()))
				whereSql := fmt.Sprintf("%v.%v = ?", scope.Quote(joinTable.joinTable), scope.Quote(ToSnake(joinTable.foreignKey)))
				scope.db.Joins(joinSql).Where(whereSql, primaryKey).Find(value)
			} else {
			}
		} else {
			association.Error = errors.New(fmt.Sprintf("invalid association %v for %v", association.Column, scopeType))
		}
	} else {
		association.Error = errors.New(fmt.Sprintf("%v doesn't have column %v", scopeType, association.Column))
	}
	return association
}

func (association *Association) Append(values interface{}) *Association {
	return association
}

func (association *Association) Delete(value interface{}) *Association {
	return association
}

func (association *Association) Clear(value interface{}) *Association {
	return association
}

func (association *Association) Replace(values interface{}) *Association {
	return association
}

func (association *Association) Count(values interface{}) int {
	return 0
}
