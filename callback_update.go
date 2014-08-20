package gorm

import (
	"fmt"
	"strings"
	"time"
)

func AssignUpdateAttributes(scope *Scope) {
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

func BeforeUpdate(scope *Scope) {
	_, ok := scope.Get("gorm:update_column")
	if !ok {
		scope.CallMethod("BeforeSave")
		scope.CallMethod("BeforeUpdate")
	}
}

func UpdateTimeStampWhenUpdate(scope *Scope) {
	_, ok := scope.Get("gorm:update_column")
	if !ok {
		scope.SetColumn("UpdatedAt", time.Now())
	}
}

func Update(scope *Scope) {
	if !scope.HasError() {
		var sqls []string

		updateAttrs, ok := scope.InstanceGet("gorm:update_attrs")
		if ok {
			for key, value := range updateAttrs.(map[string]interface{}) {
				sqls = append(sqls, fmt.Sprintf("%v = %v", scope.Quote(key), scope.AddToVars(value)))
			}
		} else {
			for _, field := range scope.Fields() {
				if field.DBName != scope.PrimaryKey() && len(field.SqlTag) > 0 && !field.IsIgnored {
					sqls = append(sqls, fmt.Sprintf("%v = %v", scope.Quote(field.DBName), scope.AddToVars(field.Value)))
				}
			}
		}

		scope.Raw(fmt.Sprintf(
			"UPDATE %v SET %v %v",
			scope.QuotedTableName(),
			strings.Join(sqls, ", "),
			scope.CombinedConditionSql(),
		))
		scope.Exec()
	}
}

func AfterUpdate(scope *Scope) {
	_, ok := scope.Get("gorm:update_column")
	if !ok {
		scope.CallMethod("AfterUpdate")
		scope.CallMethod("AfterSave")
	}
}

func init() {
	DefaultCallback.Update().Register("gorm:assign_update_attributes", AssignUpdateAttributes)
	DefaultCallback.Update().Register("gorm:begin_transaction", BeginTransaction)
	DefaultCallback.Update().Register("gorm:before_update", BeforeUpdate)
	DefaultCallback.Update().Register("gorm:save_before_associations", SaveBeforeAssociations)
	DefaultCallback.Update().Register("gorm:update_time_stamp_when_update", UpdateTimeStampWhenUpdate)
	DefaultCallback.Update().Register("gorm:update", Update)
	DefaultCallback.Update().Register("gorm:save_after_associations", SaveAfterAssociations)
	DefaultCallback.Update().Register("gorm:after_update", AfterUpdate)
	DefaultCallback.Update().Register("gorm:commit_or_rollback_transaction", CommitOrRollbackTransaction)
}
