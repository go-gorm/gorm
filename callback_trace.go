package gorm

import (
	"time"
)

// Define callbacks for tracing
func init() {
	DefaultCallback.BeforeSQL().Register("gorm:start-time", startTimeCallback)
	DefaultCallback.AfterSQL().Register("gorm:log", logCallback)
}

// startTimeCallback puts time when sql started in scope
func startTimeCallback(scope *Scope) {
	scope.Set("gorm:trace-start-time", NowFunc())
}

// logCallback prints sql log
func logCallback(scope *Scope) {
	if len(scope.SQL) <= 0 {
		return
	}

	t, ok := scope.Get("gorm:trace-start-time")
	if !ok {
		return
	}
	scope.db.slog(scope.SQL, t.(time.Time), scope.SQLVars...)
}
