package gorm

import "fmt"

// Define callbacks for deleting
func init() {
	defaultCallback.Delete().Register("gorm:begin_transaction", beginTransactionCallback)
	defaultCallback.Delete().Register("gorm:before_delete", beforeDeleteCallback)
	defaultCallback.Delete().Register("gorm:delete", deleteCallback)
	defaultCallback.Delete().Register("gorm:after_delete", afterDeleteCallback)
	defaultCallback.Delete().Register("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// beforeDeleteCallback will invoke `BeforeDelete` method before deleting
func beforeDeleteCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("BeforeDelete")
	}
}

// deleteCallback used to delete data from database or set deleted_at to current time (when using with soft delete)
func deleteCallback(scope *Scope) {
	if !scope.HasError() {
		if !scope.Search.Unscoped && scope.HasColumn("DeletedAt") {
			scope.Raw(
				fmt.Sprintf("UPDATE %v SET deleted_at=%v %v",
					scope.QuotedTableName(),
					scope.AddToVars(NowFunc()),
					scope.CombinedConditionSql(),
				)).Exec()
		} else {
			scope.Raw(fmt.Sprintf("DELETE FROM %v %v", scope.QuotedTableName(), scope.CombinedConditionSql())).Exec()
		}
	}
}

// afterDeleteCallback will invoke `AfterDelete` method after deleting
func afterDeleteCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("AfterDelete")
	}
}
