package dialect

type Dialect interface {
}

func NewDialect(driver string) *Dialect {
	var d Dialect
	switch driver {
	case "postgres":
		d = postgres{}
	case "mysql":
		d = mysql{}
	case "sqlite3":
		d = sqlite3{}
	}
	return &d
}

type mysql struct{}

type postgres struct{}

type sqlite3 struct{}
