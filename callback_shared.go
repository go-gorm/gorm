package gorm

func BeginTransaction(scope *Scope) {
	scope.Begin()
}

func CommitOrRollbackTransaction(scope *Scope) {
	scope.CommitOrRollback()
}
