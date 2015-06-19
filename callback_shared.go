package gorm

import "reflect"

func BeginTransaction(scope *Scope) {
	scope.Begin()
}

func CommitOrRollbackTransaction(scope *Scope) {
	scope.CommitOrRollback()
}

func SaveBeforeAssociations(scope *Scope) {
	if !scope.shouldSaveAssociations() {
		return
	}
	for _, field := range scope.Fields() {
		if scope.changeableField(field) && !field.IsBlank && !field.IsIgnored {
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
	if !scope.shouldSaveAssociations() {
		return
	}
	for _, field := range scope.Fields() {
		if scope.changeableField(field) && !field.IsBlank && !field.IsIgnored {
			if relationship := field.Relationship; relationship != nil &&
				(relationship.Kind == "has_one" || relationship.Kind == "has_many" || relationship.Kind == "many_to_many") {
				value := field.Field

				switch value.Kind() {
				case reflect.Slice:
					for i := 0; i < value.Len(); i++ {
						newDB := scope.NewDB()
						elem := value.Index(i).Addr().Interface()
						newScope := newDB.NewScope(elem)

						if relationship.JoinTableHandler == nil && relationship.ForeignFieldName != "" {
							scope.Err(newScope.SetColumn(relationship.ForeignFieldName, scope.PrimaryKeyValue()))
						}

						if relationship.PolymorphicType != "" {
							scope.Err(newScope.SetColumn(relationship.PolymorphicType, scope.TableName()))
						}

						scope.Err(newDB.Save(elem).Error)

						if joinTableHandler := relationship.JoinTableHandler; joinTableHandler != nil {
							scope.Err(joinTableHandler.Add(joinTableHandler, scope.NewDB(), scope.Value, newScope.Value))
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
