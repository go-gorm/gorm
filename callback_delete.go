package gorm

import (
	"fmt"
	"time"
)

func BeforeDelete(scope *Scope) {
	scope.CallMethod("BeforeDelete")
}

func Delete(scope *Scope) {
	if !scope.HasError() {
		if !scope.Search.Unscope && scope.HasColumn("DeletedAt") {
			scope.Raw(
				fmt.Sprintf("UPDATE %v SET deleted_at=%v %v",
					scope.QuotedTableName(),
					scope.AddToVars(time.Now()),
					scope.CombinedConditionSql(),
				))
		} else {
			scope.Raw(fmt.Sprintf("DELETE FROM %v %v", scope.QuotedTableName(), scope.CombinedConditionSql()))
		}

		scope.Exec()
	}
}

func AfterDelete(scope *Scope) {
	scope.CallMethod("AfterDelete")
}

func init() {
	DefaultCallback.Delete().Register("gorm:begin_transaction", BeginTransaction)
	DefaultCallback.Delete().Register("gorm:before_delete", BeforeDelete)
	DefaultCallback.Delete().Register("gorm:delete", Delete)
	DefaultCallback.Delete().Register("gorm:after_delete", AfterDelete)
	DefaultCallback.Delete().Register("gorm:commit_or_rollback_transaction", CommitOrRollbackTransaction)
}
