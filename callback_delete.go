package gorm

import "fmt"

func beforeDeleteCallback(scope *Scope) {
	scope.CallMethodWithErrorCheck("BeforeDelete")
}

func deleteCallback(scope *Scope) {
	if !scope.HasError() {
		if !scope.Search.Unscoped && scope.HasColumn("DeletedAt") {
			scope.Raw(
				fmt.Sprintf("UPDATE %v SET deleted_at=%v %v",
					scope.QuotedTableName(),
					scope.AddToVars(NowFunc()),
					scope.CombinedConditionSql(),
				))
		} else {
			scope.Raw(fmt.Sprintf("DELETE FROM %v %v", scope.QuotedTableName(), scope.CombinedConditionSql()))
		}

		scope.Exec()
	}
}

func afterDeleteCallback(scope *Scope) {
	scope.CallMethodWithErrorCheck("AfterDelete")
}

func init() {
	defaultCallback.Delete().Register("gorm:begin_transaction", beginTransactionCallback)
	defaultCallback.Delete().Register("gorm:before_delete", beforeDeleteCallback)
	defaultCallback.Delete().Register("gorm:delete", deleteCallback)
	defaultCallback.Delete().Register("gorm:after_delete", afterDeleteCallback)
	defaultCallback.Delete().Register("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}
