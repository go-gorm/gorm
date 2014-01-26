package gorm

func BeforeCreate(scope *Scope) {
	scope.CallMethod("BeforeSave")
	scope.CallMethod("BeforeCreate")
}

func SaveBeforeAssociations(scope *Scope) {
}

func Create(scope *Scope) {
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

func AfterCreate(scope *Scope) {
	scope.CallMethod("AfterCreate")
	scope.CallMethod("AfterSave")
}

func SaveAfterAssociations(scope *Scope) {
}

func init() {
	DefaultCallback.Create().Register("before_create", BeforeCreate)
	DefaultCallback.Create().Register("save_before_associations", SaveBeforeAssociations)
	DefaultCallback.Create().Register("create", Create)
	DefaultCallback.Create().Register("save_after_associations", SaveAfterAssociations)
	DefaultCallback.Create().Register("after_create", AfterCreate)
}
