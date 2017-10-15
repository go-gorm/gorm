package gorm

// DB contains information for current db connection
type DB struct {
	// Current operation
	Value  interface{} // Value current operation data
	db     SQLCommon
	search *search
	values map[string]interface{}

	// Result result fields
	Error        error
	RowsAffected int64
}
