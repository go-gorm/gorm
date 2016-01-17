package gorm

import (
	"fmt"
	"strings"
)

// Define callbacks for updating
func init() {
	defaultCallback.Update().Register("gorm:assign_updating_attributes", assignUpdatingAttributesCallback)
	defaultCallback.Update().Register("gorm:begin_transaction", beginTransactionCallback)
	defaultCallback.Update().Register("gorm:before_update", beforeUpdateCallback)
	defaultCallback.Update().Register("gorm:save_before_associations", saveBeforeAssociationsCallback)
	defaultCallback.Update().Register("gorm:update_time_stamp", updateTimeStampForUpdateCallback)
	defaultCallback.Update().Register("gorm:update", updateCallback)
	defaultCallback.Update().Register("gorm:save_after_associations", saveAfterAssociationsCallback)
	defaultCallback.Update().Register("gorm:after_update", afterUpdateCallback)
	defaultCallback.Update().Register("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// assignUpdatingAttributesCallback assign updating attributes to model
func assignUpdatingAttributesCallback(scope *Scope) {
	if attrs, ok := scope.InstanceGet("gorm:update_interface"); ok {
		if maps := convertInterfaceToMap(attrs); len(maps) > 0 {
			protected, ok := scope.Get("gorm:ignore_protected_attrs")
			_, updateColumn := scope.Get("gorm:update_column")
			updateAttrs, hasUpdate := scope.updatedAttrsWithValues(maps, ok && protected.(bool))

			if updateColumn {
				scope.InstanceSet("gorm:update_attrs", maps)
			} else if len(updateAttrs) > 0 {
				scope.InstanceSet("gorm:update_attrs", updateAttrs)
			} else if !hasUpdate {
				scope.SkipLeft()
				return
			}
		}
	}
}

// beforeUpdateCallback will invoke `BeforeSave`, `BeforeUpdate` method before updating
func beforeUpdateCallback(scope *Scope) {
	if _, ok := scope.Get("gorm:update_column"); !ok {
		if !scope.HasError() {
			scope.CallMethod("BeforeSave")
		}
		if !scope.HasError() {
			scope.CallMethod("BeforeUpdate")
		}
	}
}

// updateTimeStampForUpdateCallback will set `UpdatedAt` when updating
func updateTimeStampForUpdateCallback(scope *Scope) {
	if _, ok := scope.Get("gorm:update_column"); !ok {
		scope.SetColumn("UpdatedAt", NowFunc())
	}
}

// updateCallback the callback used to update data to database
func updateCallback(scope *Scope) {
	if !scope.HasError() {
		var sqls []string

		if updateAttrs, ok := scope.InstanceGet("gorm:update_attrs"); ok {
			for column, value := range updateAttrs.(map[string]interface{}) {
				if field, ok := scope.FieldByName(column); ok {
					if scope.changeableField(field) {
						sqls = append(sqls, fmt.Sprintf("%v = %v", scope.Quote(field.DBName), scope.AddToVars(value)))
					}
				} else {
					sqls = append(sqls, fmt.Sprintf("%v = %v", scope.Quote(column), scope.AddToVars(value)))
				}
			}
		} else {
			fields := scope.Fields()
			for _, field := range fields {
				if scope.changeableField(field) {
					if !field.IsPrimaryKey && field.IsNormal {
						sqls = append(sqls, fmt.Sprintf("%v = %v", scope.Quote(field.DBName), scope.AddToVars(field.Field.Interface())))
					} else if relationship := field.Relationship; relationship != nil && relationship.Kind == "belongs_to" {
						for _, foreignKey := range relationship.ForeignDBNames {
							if foreignField := fields[foreignKey]; !scope.changeableField(foreignField) {
								sqls = append(sqls,
									fmt.Sprintf("%v = %v", scope.Quote(foreignField.DBName), scope.AddToVars(foreignField.Field.Interface())))
							}
						}
					}
				}
			}
		}

		if len(sqls) > 0 {
			scope.Raw(fmt.Sprintf(
				"UPDATE %v SET %v %v", scope.QuotedTableName(), strings.Join(sqls, ", "), scope.CombinedConditionSql(),
			)).Exec()
		}
	}
}

// afterUpdateCallback will invoke `AfterUpdate`, `AfterSave` method after updating
func afterUpdateCallback(scope *Scope) {
	if _, ok := scope.Get("gorm:update_column"); !ok {
		if !scope.HasError() {
			scope.CallMethod("AfterUpdate")
		}
		if !scope.HasError() {
			scope.CallMethod("AfterSave")
		}
	}
}
