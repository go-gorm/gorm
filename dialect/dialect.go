package dialect

type Dialect interface {
	BinVar() string
	SupportLastInsertId() bool
	SqlTag(column interface{}, size int) string
	PrimaryKeyTag(column interface{}, size int) string
	ReturningStr(key string) string
}

func New(driver string) Dialect {
	var d Dialect
	switch driver {
	case "postgres":
		d = &postgres{}
	case "mysql":
		d = &mysql{}
	case "sqlite3":
		d = &sqlite3{}
	}
	return d
}
