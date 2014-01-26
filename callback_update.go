package gorm

func BeforeUpdate(scope *Scope) {
	scope.CallMethod("BeforeSave")
	scope.CallMethod("BeforeUpdate")
}

func Update(scope *Scope) {
	if !scope.HasError() {
		var id interface{}
		if scope.Dialect().SupportLastInsertId() {
			if sql_result, err := scope.DB().Exec(scope.Sql, scope.SqlVars...); scope.Err(err) == nil {
				id, err = sql_result.LastInsertId()
				scope.Err(err)
			}
		} else {
			scope.Err(scope.DB().QueryRow(scope.Sql, scope.SqlVars...).Scan(&id))
		}

		scope.SetColumn(scope.PrimaryKey(), id)
	}
}

func AfterUpdate(scope *Scope) {
	scope.CallMethod("AfterUpdate")
	scope.CallMethod("AfterSave")
}

func init() {
	DefaultCallback.Update().Register("before_update", BeforeUpdate)
	DefaultCallback.Update().Register("save_before_associations", SaveBeforeAssociations)
	DefaultCallback.Update().Register("update", Update)
	DefaultCallback.Update().Register("save_after_associations", SaveAfterAssociations)
	DefaultCallback.Update().Register("after_update", AfterUpdate)
}
