package gorm

import "reflect"

func beginTransactionCallback(scope *Scope) {
	scope.Begin()
}

func commitOrRollbackTransactionCallback(scope *Scope) {
	scope.CommitOrRollback()
}

func saveFieldAsAssociation(scope *Scope, field *Field) (bool, *Relationship) {
	if scope.changeableField(field) && !field.IsBlank && !field.IsIgnored {
		if value, ok := field.TagSettings["SAVE_ASSOCIATIONS"]; !ok || (value != "false" && value != "skip") {
			if relationship := field.Relationship; relationship != nil {
				return true, relationship
			}
		}
	}
	return false, nil
}

func saveBeforeAssociationsCallback(scope *Scope) {
	if !scope.shouldSaveAssociations() {
		return
	}
	for _, field := range scope.Fields() {
		if ok, relationship := saveFieldAsAssociation(scope, field); ok && relationship.Kind == "belongs_to" {
			fieldValue := field.Field.Addr().Interface()
			scope.Err(scope.NewDB().Save(fieldValue).Error)
			if len(relationship.ForeignFieldNames) != 0 {
				// set value's foreign key
				for idx, fieldName := range relationship.ForeignFieldNames {
					associationForeignName := relationship.AssociationForeignDBNames[idx]
					if foreignField, ok := scope.New(fieldValue).FieldByName(associationForeignName); ok {
						scope.Err(scope.SetColumn(fieldName, foreignField.Field.Interface()))
					}
				}
			}
		}
	}
}

func saveAfterAssociationsCallback(scope *Scope) {
	if !scope.shouldSaveAssociations() {
		return
	}
	for _, field := range scope.Fields() {
		if ok, relationship := saveFieldAsAssociation(scope, field); ok &&
			(relationship.Kind == "has_one" || relationship.Kind == "has_many" || relationship.Kind == "many_to_many") {
			value := field.Field

			switch value.Kind() {
			case reflect.Slice:
				for i := 0; i < value.Len(); i++ {
					newDB := scope.NewDB()
					elem := value.Index(i).Addr().Interface()
					newScope := newDB.NewScope(elem)

					if relationship.JoinTableHandler == nil && len(relationship.ForeignFieldNames) != 0 {
						for idx, fieldName := range relationship.ForeignFieldNames {
							associationForeignName := relationship.AssociationForeignDBNames[idx]
							if f, ok := scope.FieldByName(associationForeignName); ok {
								scope.Err(newScope.SetColumn(fieldName, f.Field.Interface()))
							}
						}
					}

					if relationship.PolymorphicType != "" {
						scope.Err(newScope.SetColumn(relationship.PolymorphicType, relationship.PolymorphicValue))
					}

					scope.Err(newDB.Save(elem).Error)

					if joinTableHandler := relationship.JoinTableHandler; joinTableHandler != nil {
						scope.Err(joinTableHandler.Add(joinTableHandler, newDB, scope.Value, newScope.Value))
					}
				}
			default:
				elem := value.Addr().Interface()
				newScope := scope.New(elem)
				if len(relationship.ForeignFieldNames) != 0 {
					for idx, fieldName := range relationship.ForeignFieldNames {
						associationForeignName := relationship.AssociationForeignDBNames[idx]
						if f, ok := scope.FieldByName(associationForeignName); ok {
							scope.Err(newScope.SetColumn(fieldName, f.Field.Interface()))
						}
					}
				}

				if relationship.PolymorphicType != "" {
					scope.Err(newScope.SetColumn(relationship.PolymorphicType, relationship.PolymorphicValue))
				}
				scope.Err(scope.NewDB().Save(elem).Error)
			}
		}
	}
}
