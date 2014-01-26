package gorm

import (
	"fmt"
	"time"
)

func BeforeDelete(scope *Scope) {
	scope.CallMethod("BeforeDelete")
}

func Delete(scope *Scope) {
	if scope.HasError() {
		return
	}

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

func AfterDelete(scope *Scope) {
	scope.CallMethod("AfterDelete")
}

func init() {
	DefaultCallback.Delete().Register("before_delete", BeforeDelete)
	DefaultCallback.Delete().Register("delete", Delete)
	DefaultCallback.Delete().Register("after_delete", AfterDelete)
}
