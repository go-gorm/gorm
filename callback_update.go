package gorm

import (
	"fmt"
	"strings"
)

func assignUpdateAttributesCallback(scope *Scope) {
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

func beforeUpdateCallback(scope *Scope) {
	if _, ok := scope.Get("gorm:update_column"); !ok {
		scope.CallMethodWithErrorCheck("BeforeSave")
		scope.CallMethodWithErrorCheck("BeforeUpdate")
	}
}

func updateTimeStampForUpdateCallback(scope *Scope) {
	if _, ok := scope.Get("gorm:update_column"); !ok {
		scope.SetColumn("UpdatedAt", NowFunc())
	}
}

func updateCallback(scope *Scope) {
	if !scope.HasError() {
		var sqls []string

		if updateAttrs, ok := scope.InstanceGet("gorm:update_attrs"); ok {
			for key, value := range updateAttrs.(map[string]interface{}) {
				if scope.changeableDBColumn(key) {
					sqls = append(sqls, fmt.Sprintf("%v = %v", scope.Quote(key), scope.AddToVars(value)))
				}
			}
		} else {
			fields := scope.Fields()
			for _, field := range fields {
				if scope.changeableField(field) && !field.IsPrimaryKey && field.IsNormal {
					sqls = append(sqls, fmt.Sprintf("%v = %v", scope.Quote(field.DBName), scope.AddToVars(field.Field.Interface())))
				} else if relationship := field.Relationship; relationship != nil && relationship.Kind == "belongs_to" {
					for _, dbName := range relationship.ForeignDBNames {
						if relationField := fields[dbName]; !scope.changeableField(relationField) && !relationField.IsBlank {
							sql := fmt.Sprintf("%v = %v", scope.Quote(relationField.DBName), scope.AddToVars(relationField.Field.Interface()))
							sqls = append(sqls, sql)
						}
					}
				}
			}
		}

		if len(sqls) > 0 {
			scope.Raw(fmt.Sprintf(
				"UPDATE %v SET %v %v",
				scope.QuotedTableName(),
				strings.Join(sqls, ", "),
				scope.CombinedConditionSql(),
			))
			scope.Exec()
		}
	}
}

func afterUpdateCallback(scope *Scope) {
	if _, ok := scope.Get("gorm:update_column"); !ok {
		scope.CallMethodWithErrorCheck("AfterUpdate")
		scope.CallMethodWithErrorCheck("AfterSave")
	}
}

func init() {
	defaultCallback.Update().Register("gorm:assign_update_attributes", assignUpdateAttributesCallback)
	defaultCallback.Update().Register("gorm:begin_transaction", beginTransactionCallback)
	defaultCallback.Update().Register("gorm:before_update", beforeUpdateCallback)
	defaultCallback.Update().Register("gorm:save_before_associations", saveBeforeAssociationsCallback)
	defaultCallback.Update().Register("gorm:update_time_stamp_when_update", updateTimeStampForUpdateCallback)
	defaultCallback.Update().Register("gorm:update", updateCallback)
	defaultCallback.Update().Register("gorm:save_after_associations", saveAfterAssociationsCallback)
	defaultCallback.Update().Register("gorm:after_update", afterUpdateCallback)
	defaultCallback.Update().Register("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}
