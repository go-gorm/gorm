package gorm

import "errors"

var (
	RecordNotFound       = errors.New("Record Not Found")
	InvalidSql           = errors.New("Invalid SQL")
	NoNewAttrs           = errors.New("No new Attributes")
	NoValidTransaction   = errors.New("No valid transaction")
	CantStartTransaction = errors.New("Can't start transaction")
)
