package gorm

func Query(scope *Scope) {
}

func AfterQuery(scope *Scope) {
	scope.CallMethod("AfterFind")
}

func init() {
	DefaultCallback.Query().Register("query", Query)
	DefaultCallback.Query().Register("after_query", AfterQuery)
}
