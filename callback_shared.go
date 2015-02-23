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
			if relationship := field.Relationship; relationship != nil && relationship.Kind == "belongs_to" {
				value := field.Field
				scope.Err(scope.NewDB().Save(value.Addr().Interface()).Error)
				if relationship.ForeignFieldName != "" {
					scope.Err(scope.SetColumn(relationship.ForeignFieldName, scope.New(value.Addr().Interface()).PrimaryKeyValue()))
				}
			}
		}
	}
}

func SaveAfterAssociations(scope *Scope) {
	for _, field := range scope.Fields() {
		if !field.IsBlank && !field.IsIgnored {
			if relationship := field.Relationship; relationship != nil &&
				(relationship.Kind == "has_one" || relationship.Kind == "has_many" || relationship.Kind == "many_to_many") {
				value := field.Field

				switch value.Kind() {
				case reflect.Slice:
					for i := 0; i < value.Len(); i++ {
						newDB := scope.NewDB()
						elem := value.Index(i).Addr().Interface()
						newScope := newDB.NewScope(elem)

						if relationship.JoinTable == "" && relationship.ForeignFieldName != "" {
							scope.Err(newScope.SetColumn(relationship.ForeignFieldName, scope.PrimaryKeyValue()))
						}

						if relationship.PolymorphicType != "" {
							scope.Err(newScope.SetColumn(relationship.PolymorphicType, scope.TableName()))
						}

						scope.Err(newDB.Save(elem).Error)

						if joinTable := relationship.JoinTable; joinTable != "" {
							quotedForeignDBName := scope.Quote(relationship.ForeignDBName)
							foreignValue := scope.PrimaryKeyValue()
							quoteAssociationForeignDBName := scope.Quote(relationship.AssociationForeignDBName)
							associationForeignValue := newScope.PrimaryKeyValue()

							newScope.Raw(fmt.Sprintf(
								"INSERT INTO %v (%v) SELECT %v %v WHERE NOT EXISTS (SELECT * FROM %v WHERE %v = %v AND %v = %v);",
								joinTable,
								strings.Join([]string{quotedForeignDBName, quoteAssociationForeignDBName}, ","),
								strings.Join([]string{newScope.AddToVars(foreignValue), newScope.AddToVars(associationForeignValue)}, ","),
								scope.Dialect().SelectFromDummyTable(),
								joinTable,
								quotedForeignDBName,
								newScope.AddToVars(foreignValue),
								quoteAssociationForeignDBName,
								newScope.AddToVars(associationForeignValue),
							))
							scope.Err(scope.NewDB().Exec(newScope.Sql, newScope.SqlVars...).Error)
						}
					}
				default:
					elem := value.Addr().Interface()
					newScope := scope.New(elem)
					if relationship.ForeignFieldName != "" {
						scope.Err(newScope.SetColumn(relationship.ForeignFieldName, scope.PrimaryKeyValue()))
					}

					if relationship.PolymorphicType != "" {
						scope.Err(newScope.SetColumn(relationship.PolymorphicType, scope.TableName()))
					}
					scope.Err(scope.NewDB().Save(elem).Error)
				}
			}
		}
	}
}
