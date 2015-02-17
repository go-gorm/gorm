package gorm

import "errors"

var (
	RecordNotFound       = errors.New("record not found")
	InvalidSql           = errors.New("invalid sql")
	NoNewAttrs           = errors.New("no new attributes")
	NoValidTransaction   = errors.New("no valid transaction")
	CantStartTransaction = errors.New("can't start transaction")
)
