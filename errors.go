package gorm

import "errors"

var (
	RecordNotFound = errors.New("Record Not Found")
	InvalidSql     = errors.New("Invalid SQL")
)
