package gorm

import "fmt"

func BeforeDelete(scope *Scope) {
	scope.CallMethodWithErrorCheck("BeforeDelete")
}

func Delete(scope *Scope) {
	if !scope.HasError() {
		unscoped := scope.Search.Unscoped
		if !unscoped && scope.HasColumn("DeletedAt") {
			scope.Raw(
				fmt.Sprintf("UPDATE %v SET deleted_at=%v %v",
					scope.QuotedTableName(),
					scope.AddToVars(NowFunc()),
					scope.CombinedConditionSql(),
				))
		} else if !unscoped && scope.HasColumn("Alive") {
			scope.Raw(
				fmt.Sprintf("UPDATE %v SET alive=null %v",
					scope.QuotedTableName(),
					scope.CombinedConditionSql(),
				))
		} else {
			scope.Raw(fmt.Sprintf("DELETE FROM %v %v", scope.QuotedTableName(), scope.CombinedConditionSql()))
		}

		scope.Exec()
	}
}

func AfterDelete(scope *Scope) {
	scope.CallMethodWithErrorCheck("AfterDelete")
}

func init() {
	DefaultCallback.Delete().Register("gorm:begin_transaction", BeginTransaction)
	DefaultCallback.Delete().Register("gorm:before_delete", BeforeDelete)
	DefaultCallback.Delete().Register("gorm:delete", Delete)
	DefaultCallback.Delete().Register("gorm:after_delete", AfterDelete)
	DefaultCallback.Delete().Register("gorm:commit_or_rollback_transaction", CommitOrRollbackTransaction)
}
