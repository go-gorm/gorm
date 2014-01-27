package gorm

import (
	"fmt"
	"strings"
	"time"
)

func AssignUpdateAttributes(scope *Scope) {
	if attrs, ok := scope.Get("gorm:update_interface"); ok {
		if maps := convertInterfaceToMap(attrs); len(maps) > 0 {
			protected, ok := scope.Get("gorm:ignore_protected_attrs")
			updateAttrs, hasUpdate := scope.updatedAttrsWithValues(maps, ok && protected.(bool))
			if len(updateAttrs) > 0 {
				scope.Set("gorm:update_attrs", updateAttrs)
			} else if !hasUpdate {
				scope.SkipLeft()
				return
			}
		}
	}
}

func BeforeUpdate(scope *Scope) {
	scope.CallMethod("BeforeSave")
	scope.CallMethod("BeforeUpdate")
}

func UpdateTimeStampWhenUpdate(scope *Scope) {
	if !scope.HasError() {
		scope.SetColumn("UpdatedAt", time.Now())
	}
}

func Update(scope *Scope) {
	defer scope.Trace(time.Now())

	if !scope.HasError() {
		var sqls []string

		updateAttrs, ok := scope.Get("gorm:update_attrs")
		if ok {
			for key, value := range updateAttrs.(map[string]interface{}) {
				sqls = append(sqls, fmt.Sprintf("%v = %v", scope.quote(key), scope.AddToVars(value)))
			}
		} else {
			for _, field := range scope.Fields() {
				if field.DBName != scope.PrimaryKey() && len(field.SqlTag) > 0 && !field.IsIgnored {
					sqls = append(sqls, fmt.Sprintf("%v = %v", scope.quote(field.DBName), scope.AddToVars(field.Value)))
				}
			}
		}

		scope.Raw(fmt.Sprintf(
			"UPDATE %v SET %v %v",
			scope.TableName(),
			strings.Join(sqls, ", "),
			scope.CombinedConditionSql(),
		))
		scope.Exec()
	}
}

func AfterUpdate(scope *Scope) {
	scope.CallMethod("AfterUpdate")
	scope.CallMethod("AfterSave")
}

func init() {
	DefaultCallback.Update().Register("assign_update_attributes", AssignUpdateAttributes)
	DefaultCallback.Update().Register("begin_transaction", BeginTransaction)
	DefaultCallback.Update().Register("before_update", BeforeUpdate)
	DefaultCallback.Update().Register("save_before_associations", SaveBeforeAssociations)
	DefaultCallback.Update().Register("update_time_stamp_when_update", UpdateTimeStampWhenUpdate)
	DefaultCallback.Update().Register("update", Update)
	DefaultCallback.Update().Register("save_after_associations", SaveAfterAssociations)
	DefaultCallback.Update().Register("after_update", AfterUpdate)
	DefaultCallback.Update().Register("commit_or_rollback_transaction", CommitOrRollbackTransaction)
}
