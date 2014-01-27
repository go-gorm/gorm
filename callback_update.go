package gorm

import (
	"fmt"
	"strings"
	"time"
)

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
	if !scope.HasError() {
		var sqls []string
		for _, field := range scope.Fields() {
			if field.DBName != scope.PrimaryKey() && len(field.SqlTag) > 0 && !field.IsIgnored {
				sqls = append(sqls, fmt.Sprintf("%v = %v", scope.quote(field.DBName), scope.AddToVars(field.Value)))
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
	DefaultCallback.Update().Register("begin_transaction", BeginTransaction)
	DefaultCallback.Update().Register("before_update", BeforeUpdate)
	DefaultCallback.Update().Register("save_before_associations", SaveBeforeAssociations)
	DefaultCallback.Update().Register("update_time_stamp_when_update", UpdateTimeStampWhenUpdate)
	DefaultCallback.Update().Register("update", Update)
	DefaultCallback.Update().Register("save_after_associations", SaveAfterAssociations)
	DefaultCallback.Update().Register("after_update", AfterUpdate)
	DefaultCallback.Update().Register("commit_or_rollback_transaction", CommitOrRollbackTransaction)
}
