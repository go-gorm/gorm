package gorm

import (
	"fmt"
	"time"
)

func BeforeDelete(scope *Scope) {
	scope.CallMethod("BeforeDelete")
}

func Delete(scope *Scope) {
	defer scope.Trace(time.Now())

	if !scope.HasError() {
		if !scope.Search.unscope && scope.HasColumn("DeletedAt") {
			scope.Raw(
				fmt.Sprintf("UPDATE %v SET deleted_at=%v %v",
					scope.TableName(),
					scope.AddToVars(time.Now()),
					scope.CombinedConditionSql(),
				))
		} else {
			scope.Raw(fmt.Sprintf("DELETE FROM %v %v", scope.TableName(), scope.CombinedConditionSql()))
		}

		scope.Exec()
	}
}

func AfterDelete(scope *Scope) {
	scope.CallMethod("AfterDelete")
}

func init() {
	DefaultCallback.Delete().Register("begin_transaction", BeginTransaction)
	DefaultCallback.Delete().Register("before_delete", BeforeDelete)
	DefaultCallback.Delete().Register("delete", Delete)
	DefaultCallback.Delete().Register("after_delete", AfterDelete)
	DefaultCallback.Delete().Register("commit_or_rollback_transaction", CommitOrRollbackTransaction)
}
