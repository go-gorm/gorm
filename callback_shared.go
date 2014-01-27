package gorm

import "reflect"

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
				newDB.Save(value.Addr().Interface())
			} else {
				// If can't take address, then clone the value and set it back
				value = reflect.New(reflect.ValueOf(field.Value).Type()).Elem()
				for _, f := range newDB.NewScope(field.Value).Fields() {
					value.FieldByName(f.Name).Set(reflect.ValueOf(f.Value))
				}
				newDB.Save(value.Addr().Interface())
				scope.SetColumn(field.Name, value.Interface())
			}

			if len(field.ForeignKey) > 0 {
				scope.SetColumn(field.ForeignKey, newDB.NewScope(value.Interface()).PrimaryKeyValue())
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

					if len(field.ForeignKey) > 0 {
						newDB.NewScope(elem).SetColumn(field.ForeignKey, scope.PrimaryKeyValue())
					}

					newDB.Save(elem)
				}
			default:
				newDB := scope.NewDB()
				if value.CanAddr() {
					newDB.NewScope(field.Value).SetColumn(field.ForeignKey, scope.PrimaryKeyValue())
					newDB.Save(field.Value)
				} else {
					destValue := reflect.New(reflect.TypeOf(field.Value)).Elem()

					for _, f := range newDB.NewScope(field.Value).Fields() {
						destValue.FieldByName(f.Name).Set(reflect.ValueOf(f.Value))
					}

					elem := destValue.Addr().Interface()
					newDB.NewScope(elem).SetColumn(field.ForeignKey, scope.PrimaryKeyValue())
					newDB.Save(elem)
					scope.SetColumn(field.Name, destValue.Interface())
				}
			}
		}
	}
}
