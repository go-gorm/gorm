package clause

import (
	"fmt"
	"log"
)

type Database string

var (
	Postgres  Database = "postgres"
	Sqlite    Database = "sqlite"
	Mysql     Database = "mysql"
	SqlServer Database = "sqlserver"
)

type Function interface {
	GetSql() string
}

type FuncDateWithoutTime struct {
	Database Database
	Field    string
	Alias    string
	SQL      string
}

func (fnc FuncDateWithoutTime) GetSql() string {
	alias := fnc.Alias
	if fnc.Alias == "" {
		alias = fnc.Field
	}
	switch fnc.Database {
	case Postgres:
		fnc.SQL = fmt.Sprintf(" to_char(%s, %s) as %s", fnc.Field, "'YYYY-MM-DD'", alias)
	case Sqlite:
		fnc.SQL = fmt.Sprintf(" strftime(%s, %s) as %s", "'%Y-%m-%d' ", fnc.Field, alias)
	case Mysql:
		fnc.SQL = fmt.Sprintf(" DATE_FORMAT(%s, %s) as %s", fnc.Field, "'%Y-%m-%d' ", alias)
	case SqlServer:
		fnc.SQL = fmt.Sprintf(" FORMAT(%s, %s) as %s", fnc.Field, "'yyyy-MM-dd' ", alias)
	default:
		log.Print("database not implemented yet for this function")
		return ""
	}

	return fnc.SQL
}
