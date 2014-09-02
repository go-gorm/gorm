package gorm

import (
	"fmt"
	"reflect"
	"strings"
)

func BeginTransaction(scope *Scope) {
	scope.Begin()
}

func CommitOrRollbackTransaction(scope *Scope) {
	scope.CommitOrRollback()
}

func SaveBeforeAssociations(scope *Scope) {
	for _, field := range scope.Fields() {
		if !field.IsBlank && !field.IsIgnored {
			relationship := field.Relationship
			if relationship != nil && relationship.Kind == "belongs_to" {
				value := field.Field
				newDB := scope.NewDB()

				if !value.CanAddr() {
					// If can't take address, then clone the value and set it back
					value = reflect.New(value.Type()).Elem()
					for _, f := range newDB.NewScope(field.Field.Interface()).Fields() {
						value.FieldByName(f.Name).Set(reflect.ValueOf(f.Field.Interface()))
					}
					scope.SetColumn(field.Name, value.Interface())
				}
				scope.Err(newDB.Save(value.Addr().Interface()).Error)

				if relationship.ForeignKey != "" {
					scope.SetColumn(relationship.ForeignKey, newDB.NewScope(value.Interface()).PrimaryKeyValue())
				}
			}
		}
	}
}

func SaveAfterAssociations(scope *Scope) {
	for _, field := range scope.Fields() {
		if !field.IsBlank && !field.IsIgnored {
			relationship := field.Relationship
			if relationship != nil &&
				(relationship.Kind == "has_one" || relationship.Kind == "has_many" || relationship.Kind == "many_to_many") {
				value := field.Field

				switch value.Kind() {
				case reflect.Slice:
					for i := 0; i < value.Len(); i++ {
						newDB := scope.NewDB()
						elem := value.Index(i).Addr().Interface()

						if relationship.JoinTable == "" && relationship.ForeignKey != "" {
							newDB.NewScope(elem).SetColumn(relationship.ForeignKey, scope.PrimaryKeyValue())
						}

						scope.Err(newDB.Save(elem).Error)

						if relationship.JoinTable != "" {
							newScope := scope.New(elem)
							joinTable := relationship.JoinTable
							foreignKey := ToSnake(relationship.ForeignKey)
							foreignValue := fmt.Sprintf("%v", scope.PrimaryKeyValue())
							associationForeignKey := ToSnake(relationship.AssociationForeignKey)
							associationForeignValue := fmt.Sprintf("%v", newScope.PrimaryKeyValue())

							newScope.Raw(fmt.Sprintf(
								"INSERT INTO %v (%v) SELECT %v %v WHERE NOT EXISTS (SELECT * FROM %v WHERE %v = %v AND %v = %v);",
								joinTable,
								strings.Join([]string{scope.Quote(foreignKey), scope.Quote(associationForeignKey)}, ","),
								strings.Join([]string{newScope.AddToVars(foreignValue), newScope.AddToVars(associationForeignValue)}, ","),
								scope.Dialect().SelectFromDummyTable(),
								joinTable,
								scope.Quote(foreignKey),
								newScope.AddToVars(foreignValue),
								scope.Quote(associationForeignKey),
								newScope.AddToVars(associationForeignValue),
							))
							scope.Err(scope.NewDB().Exec(newScope.Sql, newScope.SqlVars...).Error)
						}
					}
				default:
					newDB := scope.NewDB()
					if value.CanAddr() {
						if relationship.ForeignKey != "" {
							newDB.NewScope(value.Addr().Interface()).SetColumn(relationship.ForeignKey, scope.PrimaryKeyValue())
						}
						scope.Err(newDB.Save(value.Addr().Interface()).Error)
					} else {
						destValue := reflect.New(field.Field.Type()).Elem()

						for _, f := range newDB.NewScope(field.Field.Interface()).Fields() {
							destValue.FieldByName(f.Name).Set(f.Field)
						}

						elem := destValue.Addr().Interface()
						if relationship.ForeignKey != "" {
							newDB.NewScope(elem).SetColumn(relationship.ForeignKey, scope.PrimaryKeyValue())
						}
						scope.Err(newDB.Save(elem).Error)
						scope.SetColumn(field.Name, destValue.Interface())
					}
				}
			}
		}
	}
}
