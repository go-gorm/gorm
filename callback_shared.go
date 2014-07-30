package gorm

import (
	"fmt"
	"reflect"
)

func BeginTransaction(scope *Scope) {
	scope.Begin()
}

func CommitOrRollbackTransaction(scope *Scope) {
	scope.CommitOrRollback()
}

func SaveBeforeAssociations(scope *Scope) {
	for _, field := range scope.Fields() {
		if field.BeforeAssociation && !field.IsBlank && !field.IsIgnored {
			value := reflect.ValueOf(field.Value)
			newDB := scope.NewDB()

			if value.CanAddr() {
				scope.Err(newDB.Save(value.Addr().Interface()).Error)
			} else {
				// If can't take address, then clone the value and set it back
				value = reflect.New(reflect.ValueOf(field.Value).Type()).Elem()
				for _, f := range newDB.NewScope(field.Value).Fields() {
					value.FieldByName(f.Name).Set(reflect.ValueOf(f.Value))
				}
				scope.Err(newDB.Save(value.Addr().Interface()).Error)
				scope.SetColumn(field.Name, value.Interface())
			}

			if field.JoinTable != nil && field.JoinTable.foreignKey != "" {
				scope.SetColumn(field.JoinTable.foreignKey, newDB.NewScope(value.Interface()).PrimaryKeyValue())
			}
		}
	}
}

func SaveAfterAssociations(scope *Scope) {
	for _, field := range scope.Fields() {
		if field.AfterAssociation && !field.IsBlank && !field.IsIgnored {
			value := reflect.ValueOf(field.Value)

			switch value.Kind() {
			case reflect.Slice:
				for i := 0; i < value.Len(); i++ {
					newDB := scope.NewDB()
					elem := value.Index(i).Addr().Interface()

					if field.JoinTable != nil && field.JoinTable.foreignKey != "" {
						newDB.NewScope(elem).SetColumn(field.JoinTable.foreignKey, scope.PrimaryKeyValue())
					}

					scope.Err(newDB.Save(elem).Error)
					fmt.Sprintf("INSERT INTO %v (%v, %v) SELECT (%v, %v) FROM %v WHERE NOT EXISTS (SELECT * FROM %v WHERE %v = %v AND %v = %v) limit 1;")
				}
			default:
				newDB := scope.NewDB()
				if value.CanAddr() {
					if field.JoinTable != nil {
						newDB.NewScope(field.Value).SetColumn(field.JoinTable.foreignKey, scope.PrimaryKeyValue())
					}
					scope.Err(newDB.Save(field.Value).Error)
				} else {
					destValue := reflect.New(reflect.TypeOf(field.Value)).Elem()

					for _, f := range newDB.NewScope(field.Value).Fields() {
						destValue.FieldByName(f.Name).Set(reflect.ValueOf(f.Value))
					}

					elem := destValue.Addr().Interface()
					if field.JoinTable != nil {
						newDB.NewScope(elem).SetColumn(field.JoinTable.foreignKey, scope.PrimaryKeyValue())
					}
					scope.Err(newDB.Save(elem).Error)
					scope.SetColumn(field.Name, destValue.Interface())
				}
			}
		}
	}
}
